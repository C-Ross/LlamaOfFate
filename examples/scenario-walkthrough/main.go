package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	gameconfig "github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

func main() {
	logging.SetupDefaultLogging()

	// Scenario source flags (pick one)
	scenarioFlag := flag.String("scenario", "", "Predefined scenario: saloon, heist, tower")
	genreFlag := flag.String("genre", "", "Genre for generated scenario (e.g., Western, Cyberpunk, Fantasy)")

	// Player character flags
	nameFlag := flag.String("name", "Jesse Calhoun", "Player character name")
	conceptFlag := flag.String("concept", "Haunted Former Rancher Seeking Justice", "Player high concept")
	troubleFlag := flag.String("trouble", "Vengeance Burns Hotter Than Reason", "Player trouble aspect")

	// Output flags
	logFlag := flag.String("log", "auto", "Session log path (auto=generated filename, empty=disabled)")
	maxScenesFlag := flag.Int("max-scenes", 10, "Maximum number of scenes before stopping")

	// Debug flags
	debugFlag := flag.Bool("debug", false, "Show rendered prompts")
	rawFlag := flag.Bool("raw", false, "Show raw LLM responses")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: scenario-walkthrough [options]\n\n")
		fmt.Fprintf(os.Stderr, "Walk through an entire Fate Core scenario by narrating scene outcomes.\n")
		fmt.Fprintf(os.Stderr, "Tests the full chain: scenario -> scenes -> summaries -> resolution.\n\n")
		fmt.Fprintf(os.Stderr, "Scenario Source (pick one):\n")
		fmt.Fprintf(os.Stderr, "  -scenario string   Predefined scenario (saloon, heist, tower)\n")
		fmt.Fprintf(os.Stderr, "  -genre string      Generate scenario for genre (Western, Cyberpunk, Fantasy)\n\n")
		fmt.Fprintf(os.Stderr, "Player Character:\n")
		fmt.Fprintf(os.Stderr, "  -name string       Character name (default: Jesse Calhoun)\n")
		fmt.Fprintf(os.Stderr, "  -concept string    High concept aspect\n")
		fmt.Fprintf(os.Stderr, "  -trouble string    Trouble aspect\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  -log string        Session log path (default: auto-generated, empty disables)\n")
		fmt.Fprintf(os.Stderr, "  -max-scenes int    Maximum scenes before stopping (default: 10)\n\n")
		fmt.Fprintf(os.Stderr, "Debug:\n")
		fmt.Fprintf(os.Stderr, "  -debug             Show rendered prompts\n")
		fmt.Fprintf(os.Stderr, "  -raw               Show raw LLM responses\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  scenario-walkthrough -scenario saloon\n")
		fmt.Fprintf(os.Stderr, "  scenario-walkthrough -genre Western -name \"Jesse Calhoun\" \\\n")
		fmt.Fprintf(os.Stderr, "    -concept \"Haunted Former Rancher\" -trouble \"Vengeance Burns Hotter Than Reason\"\n")
		fmt.Fprintf(os.Stderr, "  scenario-walkthrough -genre Cyberpunk -name Nova-7 -concept \"Rogue AI\"\n")
	}

	flag.Parse()

	if *scenarioFlag == "" && *genreFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: provide -scenario or -genre\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load Azure config
	configPath := "configs/azure-llm.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Azure LLM config not found at %s\n", configPath)
		fmt.Println("Please copy configs/azure-llm.yaml.example to configs/azure-llm.yaml")
		fmt.Println("and configure your Azure OpenAI credentials.")
		os.Exit(1)
	}

	config, err := azure.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load Azure config: %v", err)
	}

	azureClient := azure.NewClient(*config)

	// Set up session logging
	var sessionLogger *session.Logger
	logPath := *logFlag
	if logPath == "auto" {
		label := *scenarioFlag
		if label == "" {
			label = *genreFlag
		}
		var err error
		logPath, err = session.GenerateLogPath("walkthrough", []string{label, *nameFlag}, 20)
		if err != nil {
			log.Fatalf("Failed to generate log path: %v", err)
		}
	}
	if logPath != "" {
		sessionLogger, err = session.NewLogger(logPath)
		if err != nil {
			log.Fatalf("Failed to create session logger: %v", err)
		}
		defer func() {
			if err := sessionLogger.Close(); err != nil {
				log.Printf("Warning: Failed to close session logger: %v", err)
			}
		}()
		fmt.Printf("Session log: %s\n\n", logPath)
	}

	ctx := context.Background()

	// Step 1: Get or generate the scenario
	var scenario *scene.Scenario
	if *scenarioFlag != "" {
		scenarios, loadErr := gameconfig.LoadAll("configs")
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to load configs: %v\n", loadErr)
			os.Exit(1)
		}
		ls, ok := scenarios[*scenarioFlag]
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown scenario: %s (use saloon, heist, or tower)\n", *scenarioFlag)
			os.Exit(1)
		}
		scenario = ls.Scenario
		fmt.Println("=== Using Predefined Scenario ===")
	} else {
		fmt.Println("=== Generating Scenario ===")
		scenario, err = generateScenario(ctx, azureClient, *nameFlag, *conceptFlag, *troubleFlag, *genreFlag, *debugFlag, *rawFlag, sessionLogger)
		if err != nil {
			log.Fatalf("Failed to generate scenario: %v", err)
		}
	}

	displayScenario(scenario)

	if sessionLogger != nil {
		sessionLogger.Log("scenario", scenario)
	}

	// Step 2: Walk through scenes
	reader := bufio.NewReader(os.Stdin)
	var sceneSummaries []prompt.SceneSummary
	transitionHint := ""
	sceneCount := 0

	for sceneCount < *maxScenesFlag {
		sceneCount++

		fmt.Printf("\n=== Generating Scene %d ===\n", sceneCount)
		generated, err := generateScene(ctx, azureClient, transitionHint, scenario, *nameFlag, *conceptFlag, *troubleFlag, sceneSummaries, *debugFlag, *rawFlag, sessionLogger)
		if err != nil {
			log.Fatalf("Failed to generate scene %d: %v", sceneCount, err)
		}

		displayScene(sceneCount, generated)

		fmt.Println("What happens in this scene? (or 'quit')")
		fmt.Print("> ")
		narration, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}
		narration = strings.TrimSpace(narration)

		if strings.ToLower(narration) == "quit" || strings.ToLower(narration) == "q" {
			fmt.Println("\n=== Walkthrough Ended (player quit) ===")
			if sessionLogger != nil {
				sessionLogger.Log("walkthrough_end", map[string]any{
					"reason":      "quit",
					"scene_count": sceneCount,
				})
			}
			break
		}

		if sessionLogger != nil {
			sessionLogger.Log("player_narration", map[string]any{
				"scene_number": sceneCount,
				"scene_name":   generated.SceneName,
				"narration":    narration,
			})
		}

		transitionHint = extractTransitionHint(narration)

		fmt.Printf("\n--- Generating Scene %d Summary ---\n", sceneCount)
		summary, err := generateSummaryFromNarration(ctx, azureClient, generated, narration, transitionHint, *debugFlag, *rawFlag, sessionLogger)
		if err != nil {
			fmt.Printf("Warning: Failed to generate summary: %v\n", err)
			fmt.Println("Continuing without summary...")
		} else {
			sceneSummaries = appendSummary(sceneSummaries, summary)
			displaySummary(summary)

			if sessionLogger != nil {
				sessionLogger.Log("scene_summary", map[string]any{
					"scene_number": sceneCount,
					"summary":      summary,
				})
			}
		}

		// Check scenario resolution
		if len(scenario.StoryQuestions) > 0 && len(sceneSummaries) > 0 {
			fmt.Printf("\n--- Resolution Check ---\n")
			resolution, err := checkResolution(ctx, azureClient, scenario, sceneSummaries, *nameFlag, *conceptFlag, *troubleFlag, *debugFlag, *rawFlag, sessionLogger)
			if err != nil {
				fmt.Printf("Warning: Failed to check resolution: %v\n", err)
			} else {
				displayResolution(resolution)

				if sessionLogger != nil {
					sessionLogger.Log("resolution_check", map[string]any{
						"scene_number": sceneCount,
						"resolution":   resolution,
					})
				}

				if resolution.IsResolved {
					fmt.Println("\n=== Scenario Resolved! ===")
					fmt.Printf("Completed in %d scene(s).\n", sceneCount)
					if sessionLogger != nil {
						sessionLogger.Log("walkthrough_end", map[string]any{
							"reason":      "resolved",
							"scene_count": sceneCount,
						})
					}
					break
				}
			}
		}

		if sceneCount >= *maxScenesFlag {
			fmt.Printf("\n=== Walkthrough Ended (reached max %d scenes) ===\n", *maxScenesFlag)
			if sessionLogger != nil {
				sessionLogger.Log("walkthrough_end", map[string]any{
					"reason":      "max_scenes",
					"scene_count": sceneCount,
				})
			}
		}
	}

	// Final summary
	fmt.Println()
	fmt.Println("=== Walkthrough Summary ===")
	fmt.Printf("Scenario: %s\n", scenario.Title)
	fmt.Printf("Scenes Played: %d\n", sceneCount)
	if len(sceneSummaries) > 0 {
		fmt.Println("\nScene Recaps:")
		for i, s := range sceneSummaries {
			fmt.Printf("  %d. %s\n", i+1, s.NarrativeProse)
		}
	}
}

// generateScenario creates a scenario via LLM from character aspects
func generateScenario(ctx context.Context, client llm.LLMClient, name, concept, trouble, genre string, debug, raw bool, logger *session.Logger) (*scene.Scenario, error) {
	data := prompt.ScenarioGenerationData{
		PlayerName:        name,
		PlayerHighConcept: concept,
		PlayerTrouble:     trouble,
		Genre:             genre,
	}

	promptText, err := prompt.RenderScenarioGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("scenario_generation_prompt", map[string]any{"prompt": promptText})
	}

	if debug {
		fmt.Println("--- Scenario Generation Prompt ---")
		fmt.Println(promptText)
		fmt.Println()
	}

	rawResponse, err := llm.SimpleCompletion(ctx, client, promptText, 500, 0.8)
	if err != nil {
		return nil, err
	}
	if raw {
		fmt.Println("--- Raw Scenario Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return prompt.ParseScenario(rawResponse)
}

// generateScene creates a scene via LLM
func generateScene(ctx context.Context, client llm.LLMClient, hint string, scenario *scene.Scenario, name, concept, trouble string, summaries []prompt.SceneSummary, debug, raw bool, logger *session.Logger) (*prompt.GeneratedScene, error) {
	data := prompt.SceneGenerationData{
		TransitionHint:    hint,
		Scenario:          scenario,
		PlayerName:        name,
		PlayerHighConcept: concept,
		PlayerTrouble:     trouble,
		PreviousSummaries: summaries,
	}

	promptText, err := prompt.RenderSceneGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("scene_generation_prompt", map[string]any{"prompt": promptText})
	}

	if debug {
		fmt.Println("--- Scene Generation Prompt ---")
		fmt.Println(promptText)
		fmt.Println()
	}

	rawResponse, err := llm.SimpleCompletion(ctx, client, promptText, 500, 0.8)
	if err != nil {
		return nil, err
	}
	if raw {
		fmt.Println("--- Raw Scene Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return prompt.ParseGeneratedScene(rawResponse)
}

// generateSummaryFromNarration creates a scene summary using player narration as the conversation history
func generateSummaryFromNarration(ctx context.Context, client llm.LLMClient, scene *prompt.GeneratedScene, narration, transitionHint string, debug, raw bool, logger *session.Logger) (*prompt.SceneSummary, error) {
	var aspects []string
	aspects = append(aspects, scene.SituationAspects...)

	var npcs []prompt.NPCSummary
	for _, npc := range scene.NPCs {
		npcs = append(npcs, prompt.NPCSummary{
			Name:     npc.Name,
			Attitude: npc.Disposition,
		})
	}

	data := prompt.SceneSummaryData{
		SceneName:        scene.SceneName,
		SceneDescription: scene.Description,
		SituationAspects: aspects,
		ConversationHistory: []prompt.ConversationEntry{
			{
				PlayerInput: narration,
				GMResponse:  "(Narrated outcome -- walkthrough mode)",
			},
		},
		NPCsInScene:    npcs,
		HowEnded:       "transition",
		TransitionHint: transitionHint,
	}

	promptText, err := prompt.RenderSceneSummary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("summary_generation_prompt", map[string]any{"prompt": promptText})
	}

	if debug {
		fmt.Println("--- Summary Generation Prompt ---")
		fmt.Println(promptText)
		fmt.Println()
	}

	rawResponse, err := llm.SimpleCompletion(ctx, client, promptText, 400, 0.5)
	if err != nil {
		return nil, err
	}
	if raw {
		fmt.Println("--- Raw Summary Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return prompt.ParseSceneSummary(rawResponse)
}

// checkResolution checks if the scenario's story questions have been answered
func checkResolution(ctx context.Context, client llm.LLMClient, scenario *scene.Scenario, summaries []prompt.SceneSummary, name, concept, trouble string, debug, raw bool, logger *session.Logger) (*prompt.ScenarioResolutionResult, error) {
	var playerAspects []string
	playerAspects = append(playerAspects, concept)
	if trouble != "" {
		playerAspects = append(playerAspects, trouble)
	}

	var latestSummary *prompt.SceneSummary
	if len(summaries) > 0 {
		latestSummary = &summaries[len(summaries)-1]
	}

	data := prompt.ScenarioResolutionData{
		Scenario:       scenario,
		SceneSummaries: summaries,
		LatestSummary:  latestSummary,
		PlayerName:     name,
		PlayerAspects:  playerAspects,
	}

	promptText, err := prompt.RenderScenarioResolution(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("resolution_check_prompt", map[string]any{"prompt": promptText})
	}

	if debug {
		fmt.Println("--- Resolution Check Prompt ---")
		fmt.Println(promptText)
		fmt.Println()
	}

	rawResponse, err := llm.SimpleCompletion(ctx, client, promptText, 300, 0.3)
	if err != nil {
		return nil, err
	}
	if raw {
		fmt.Println("--- Raw Resolution Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return prompt.ParseScenarioResolution(rawResponse)
}

// extractTransitionHint tries to extract a destination from narration
func extractTransitionHint(narration string) string {
	lower := strings.ToLower(narration)

	prefixes := []string{
		"heads to ", "goes to ", "moves to ", "travels to ",
		"rides to ", "walks to ", "runs to ", "heads toward ",
		"heads towards ", "makes for ", "leaves for ",
	}
	for _, prefix := range prefixes {
		if idx := strings.Index(lower, prefix); idx >= 0 {
			hint := narration[idx+len(prefix):]
			hint = strings.TrimRight(hint, ".!,;")
			if hint != "" {
				return hint
			}
		}
	}

	sentences := strings.Split(narration, ".")
	for i := len(sentences) - 1; i >= 0; i-- {
		s := strings.TrimSpace(sentences[i])
		if s != "" {
			return s
		}
	}

	return narration
}

// appendSummary adds a summary and keeps a sliding window of last 3
func appendSummary(summaries []prompt.SceneSummary, summary *prompt.SceneSummary) []prompt.SceneSummary {
	if summary == nil {
		return summaries
	}
	summaries = append(summaries, *summary)
	if len(summaries) > 3 {
		summaries = summaries[len(summaries)-3:]
	}
	return summaries
}

// === Display functions ===

func displayScenario(s *scene.Scenario) {
	fmt.Printf("Title: %s\n", s.Title)
	if s.Genre != "" {
		fmt.Printf("Genre: %s\n", s.Genre)
	}
	fmt.Println()
	fmt.Printf("Problem: %s\n", s.Problem)
	fmt.Println()
	if len(s.StoryQuestions) > 0 {
		fmt.Println("Story Questions:")
		for i, q := range s.StoryQuestions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()
	}
	if s.Setting != "" {
		fmt.Println("Setting:")
		printWrapped(s.Setting, 75, "  ")
		fmt.Println()
	}
}

func displayScene(num int, s *prompt.GeneratedScene) {
	fmt.Printf("\n=== Scene %d: %s ===\n", num, s.SceneName)
	fmt.Println()

	if s.Purpose != "" {
		fmt.Println("Purpose:")
		printWrapped(s.Purpose, 75, "  ")
		fmt.Println()
	}

	fmt.Println("Description:")
	printWrapped(s.Description, 75, "  ")
	fmt.Println()

	if s.OpeningHook != "" {
		fmt.Println("Opening Hook:")
		printWrapped(s.OpeningHook, 75, "  ")
		fmt.Println()
	}

	if len(s.SituationAspects) > 0 {
		fmt.Println("Aspects:")
		for _, a := range s.SituationAspects {
			fmt.Printf("  - \"%s\"\n", a)
		}
	}

	if len(s.NPCs) > 0 {
		fmt.Println("NPCs:")
		for _, npc := range s.NPCs {
			fmt.Printf("  - %s (%s) [%s]\n", npc.Name, npc.HighConcept, npc.Disposition)
		}
	}
	fmt.Println()
}

func displaySummary(s *prompt.SceneSummary) {
	fmt.Println()
	if len(s.KeyEvents) > 0 {
		fmt.Println("Key Events:")
		for _, e := range s.KeyEvents {
			fmt.Printf("  - %s\n", e)
		}
	}
	if len(s.NPCsEncountered) > 0 {
		fmt.Println("NPCs:")
		for _, n := range s.NPCsEncountered {
			fmt.Printf("  - %s (%s)\n", n.Name, n.Attitude)
		}
	}
	if len(s.UnresolvedThreads) > 0 {
		fmt.Println("Unresolved:")
		for _, t := range s.UnresolvedThreads {
			fmt.Printf("  - %s\n", t)
		}
	}
	fmt.Println()
	fmt.Printf("Recap: %s\n", s.NarrativeProse)
}

func displayResolution(r *prompt.ScenarioResolutionResult) {
	if len(r.AnsweredQuestions) > 0 {
		fmt.Println("Answered Questions:")
		for _, q := range r.AnsweredQuestions {
			fmt.Printf("  - %s\n", q)
		}
	}
	fmt.Printf("Reasoning: %s\n", r.Reasoning)
	if r.IsResolved {
		fmt.Println("Status: RESOLVED")
	} else {
		fmt.Println("Status: Not yet resolved")
	}
}

func printWrapped(text string, width int, indent string) {
	words := strings.Fields(text)
	line := indent
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			fmt.Println(line)
			line = indent + word
		} else {
			if line == indent {
				line += word
			} else {
				line += " " + word
			}
		}
	}
	if line != indent {
		fmt.Println(line)
	}
}
