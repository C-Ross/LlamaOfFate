package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
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
			label = strings.ToLower(*genreFlag)
		}
		safeName := strings.ToLower(strings.ReplaceAll(*nameFlag, " ", "_"))
		safeName = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
				return r
			}
			return -1
		}, safeName)
		if len(safeName) > 20 {
			safeName = safeName[:20]
		}
		logPath = fmt.Sprintf("walkthrough_%s_%s_%s.yaml", label, safeName, time.Now().Format("20060102_150405"))
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
	var scenario *engine.Scenario
	if *scenarioFlag != "" {
		scenario = getScenario(*scenarioFlag)
		if scenario == nil {
			fmt.Fprintf(os.Stderr, "Unknown scenario: %s (use saloon, heist, or tower)\n", *scenarioFlag)
			os.Exit(1)
		}
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
	var sceneSummaries []engine.SceneSummary
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
func generateScenario(ctx context.Context, client llm.LLMClient, name, concept, trouble, genre string, debug, raw bool, logger *session.Logger) (*engine.Scenario, error) {
	data := engine.ScenarioGenerationData{
		PlayerName:        name,
		PlayerHighConcept: concept,
		PlayerTrouble:     trouble,
		Genre:             genre,
	}

	prompt, err := engine.RenderScenarioGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("scenario_generation_prompt", map[string]any{"prompt": prompt})
	}

	if debug {
		fmt.Println("--- Scenario Generation Prompt ---")
		fmt.Println(prompt)
		fmt.Println()
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	rawResponse := resp.Choices[0].Message.Content
	if raw {
		fmt.Println("--- Raw Scenario Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return parseScenario(rawResponse)
}

// generateScene creates a scene via LLM
func generateScene(ctx context.Context, client llm.LLMClient, hint string, scenario *engine.Scenario, name, concept, trouble string, summaries []engine.SceneSummary, debug, raw bool, logger *session.Logger) (*engine.GeneratedScene, error) {
	data := engine.SceneGenerationData{
		TransitionHint:    hint,
		Scenario:          scenario,
		PlayerName:        name,
		PlayerHighConcept: concept,
		PlayerTrouble:     trouble,
		PreviousSummaries: summaries,
	}

	prompt, err := engine.RenderSceneGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("scene_generation_prompt", map[string]any{"prompt": prompt})
	}

	if debug {
		fmt.Println("--- Scene Generation Prompt ---")
		fmt.Println(prompt)
		fmt.Println()
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	rawResponse := resp.Choices[0].Message.Content
	if raw {
		fmt.Println("--- Raw Scene Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return parseGeneratedScene(rawResponse)
}

// generateSummaryFromNarration creates a scene summary using player narration as the conversation history
func generateSummaryFromNarration(ctx context.Context, client llm.LLMClient, scene *engine.GeneratedScene, narration, transitionHint string, debug, raw bool, logger *session.Logger) (*engine.SceneSummary, error) {
	var aspects []string
	aspects = append(aspects, scene.SituationAspects...)

	var npcs []engine.NPCSummary
	for _, npc := range scene.NPCs {
		npcs = append(npcs, engine.NPCSummary{
			Name:     npc.Name,
			Attitude: npc.Disposition,
		})
	}

	data := engine.SceneSummaryData{
		SceneName:        scene.SceneName,
		SceneDescription: scene.Description,
		SituationAspects: aspects,
		ConversationHistory: []engine.ConversationEntry{
			{
				PlayerInput: narration,
				GMResponse:  "(Narrated outcome -- walkthrough mode)",
			},
		},
		NPCsInScene:    npcs,
		HowEnded:       "transition",
		TransitionHint: transitionHint,
	}

	prompt, err := engine.RenderSceneSummary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("summary_generation_prompt", map[string]any{"prompt": prompt})
	}

	if debug {
		fmt.Println("--- Summary Generation Prompt ---")
		fmt.Println(prompt)
		fmt.Println()
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   400,
		Temperature: 0.5,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	rawResponse := resp.Choices[0].Message.Content
	if raw {
		fmt.Println("--- Raw Summary Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return parseSceneSummary(rawResponse)
}

// checkResolution checks if the scenario's story questions have been answered
func checkResolution(ctx context.Context, client llm.LLMClient, scenario *engine.Scenario, summaries []engine.SceneSummary, name, concept, trouble string, debug, raw bool, logger *session.Logger) (*engine.ScenarioResolutionResult, error) {
	var playerAspects []string
	playerAspects = append(playerAspects, concept)
	if trouble != "" {
		playerAspects = append(playerAspects, trouble)
	}

	var latestSummary *engine.SceneSummary
	if len(summaries) > 0 {
		latestSummary = &summaries[len(summaries)-1]
	}

	data := engine.ScenarioResolutionData{
		Scenario:       scenario,
		SceneSummaries: summaries,
		LatestSummary:  latestSummary,
		PlayerName:     name,
		PlayerAspects:  playerAspects,
	}

	prompt, err := engine.RenderScenarioResolution(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	if logger != nil {
		logger.Log("resolution_check_prompt", map[string]any{"prompt": prompt})
	}

	if debug {
		fmt.Println("--- Resolution Check Prompt ---")
		fmt.Println(prompt)
		fmt.Println()
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   300,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	rawResponse := resp.Choices[0].Message.Content
	if raw {
		fmt.Println("--- Raw Resolution Response ---")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	return parseResolution(rawResponse)
}

// === Parsing functions ===

func parseScenario(content string) (*engine.Scenario, error) {
	cleaned := cleanJSONResponse(content)

	var scenario engine.Scenario
	if err := json.Unmarshal([]byte(cleaned), &scenario); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &scenario); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if scenario.Title == "" {
		return nil, fmt.Errorf("missing title")
	}
	if scenario.Problem == "" {
		return nil, fmt.Errorf("missing problem")
	}

	return &scenario, nil
}

func parseGeneratedScene(content string) (*engine.GeneratedScene, error) {
	cleaned := cleanJSONResponse(content)

	var generated engine.GeneratedScene
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if generated.SceneName == "" {
		return nil, fmt.Errorf("missing scene_name")
	}
	if generated.Description == "" {
		return nil, fmt.Errorf("missing description")
	}

	return &generated, nil
}

func parseSceneSummary(content string) (*engine.SceneSummary, error) {
	cleaned := cleanJSONResponse(content)

	var summary engine.SceneSummary
	if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if summary.NarrativeProse == "" {
		return nil, fmt.Errorf("missing narrative_prose")
	}

	return &summary, nil
}

func parseResolution(content string) (*engine.ScenarioResolutionResult, error) {
	cleaned := cleanJSONResponse(content)

	var result engine.ScenarioResolutionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	return &result, nil
}

// cleanJSONResponse removes markdown formatting from LLM JSON responses
func cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)

	blocks := strings.Split(content, "```")
	var jsonBlocks []string

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if strings.HasPrefix(block, "json\n") {
			block = strings.TrimPrefix(block, "json\n")
			block = strings.TrimSpace(block)
			if strings.HasPrefix(block, "{") && strings.HasSuffix(block, "}") {
				jsonBlocks = append(jsonBlocks, block)
			}
		} else if strings.HasPrefix(block, "{") && strings.HasSuffix(block, "}") {
			jsonBlocks = append(jsonBlocks, block)
		}
	}

	if len(jsonBlocks) > 0 {
		return jsonBlocks[len(jsonBlocks)-1]
	}

	return content
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
func appendSummary(summaries []engine.SceneSummary, summary *engine.SceneSummary) []engine.SceneSummary {
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

func displayScenario(s *engine.Scenario) {
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

func displayScene(num int, s *engine.GeneratedScene) {
	fmt.Printf("\n=== Scene %d: %s ===\n", num, s.SceneName)
	fmt.Println()
	fmt.Println("Description:")
	printWrapped(s.Description, 75, "  ")
	fmt.Println()

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

func displaySummary(s *engine.SceneSummary) {
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

func displayResolution(r *engine.ScenarioResolutionResult) {
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

// getScenario returns a predefined scenario by name
func getScenario(name string) *engine.Scenario {
	switch strings.ToLower(name) {
	case "tower":
		return &engine.Scenario{
			Title:   "The Wizard's Tower",
			Problem: "A mysterious magical disturbance threatens the tower and its inhabitants",
			StoryQuestions: []string{
				"Can the source of the disturbance be discovered?",
				"Will the tower's secrets be revealed?",
			},
			Genre:   "Fantasy",
			Setting: "A medieval fantasy world of magic and mystery. Wizards study arcane arts in towers, adventurers seek treasure in ancient ruins, and supernatural forces are very real.",
		}
	case "heist":
		return &engine.Scenario{
			Title:   "The Prometheus Job",
			Problem: "A high-value data core must be extracted from a heavily guarded corporate facility",
			StoryQuestions: []string{
				"Can the team breach the facility's security?",
				"Will the extraction succeed without casualties?",
				"What secrets does the data core contain?",
			},
			Genre:   "Cyberpunk",
			Setting: "A dark near-future where megacorporations rule, hackers breach digital fortresses, and chrome-enhanced mercenaries sell their skills to the highest bidder. Neon lights flicker over rain-slicked streets.",
		}
	case "saloon":
		return &engine.Scenario{
			Title:   "Trouble in Redemption Gulch",
			Problem: "The town is under threat from outlaws and someone needs to stand up for the innocent",
			StoryQuestions: []string{
				"Will the outlaws be brought to justice?",
				"Can the town be saved?",
			},
			Genre:   "Western",
			Setting: "The American Old West in the late 1800s. Dusty frontier towns, lawless territories, and the struggle between civilization and the wild. Gunslingers, outlaws, and honest folk all seeking their fortune.",
		}
	default:
		return nil
	}
}
