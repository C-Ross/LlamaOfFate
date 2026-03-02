package prompt

import (
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// Template data types for LLM prompt generation.
// These structs are passed to Go templates to render prompts.

// DifficultyGuidance holds pre-computed difficulty thresholds derived from
// a character's peak skill. Both the action parser and challenge generator
// embed this so the LLM picks sensible difficulties.
type DifficultyGuidance struct {
	DifficultyMin     int    // Recommended minimum difficulty
	DifficultyMax     int    // Recommended maximum difficulty
	DifficultyDefault int    // Suggested default difficulty
	DifficultyGuide   string // Human-readable difficulty guidance
}

// ComputeDifficultyGuidance derives difficulty thresholds from a character's
// skill map. The range follows Fate Core heuristics: 2 below peak = easy,
// at peak = moderate, 2 above peak = hard.
func ComputeDifficultyGuidance(skills map[string]dice.Ladder) DifficultyGuidance {
	highestSkill := dice.Mediocre
	for _, level := range skills {
		if level > highestSkill {
			highestSkill = level
		}
	}

	minDiff := int(dice.Average)     // Floor at Average (+1) for meaningful rolls
	defaultDiff := int(dice.Fair)    // Fair (+2) is the standard default
	maxDiff := int(highestSkill) + 2 // Up to 2 above their best skill
	if maxDiff < int(dice.Good) {
		maxDiff = int(dice.Good) // At least Good (+3) as upper bound
	}
	if maxDiff > int(dice.Fantastic) {
		maxDiff = int(dice.Fantastic) // Cap at Fantastic (+6) for normal play
	}

	guide := fmt.Sprintf("%d=easy, %d=moderate, %d=hard", minDiff, defaultDiff, maxDiff)

	return DifficultyGuidance{
		DifficultyMin:     minDiff,
		DifficultyMax:     maxDiff,
		DifficultyDefault: defaultDiff,
		DifficultyGuide:   guide,
	}
}

// ChallengeBuildData holds context for the challenge build prompt, which
// generates structured task lists from a challenge description. This is
// the second LLM call in the two-step challenge flow: (1) the scene
// response LLM emits [CHALLENGE:description], (2) this prompt turns that
// description into concrete tasks with skills and difficulties.
type ChallengeBuildData struct {
	Description        string                  // Challenge description from the marker
	SceneName          string                  // Current scene name for context
	SceneDescription   string                  // Current scene description
	PlayerSkills       []string                // Player's skill names (so tasks use real skills)
	SituationAspects   []scene.SituationAspect // Active scene aspects for narrative context
	DifficultyGuidance                         // Embedded difficulty thresholds
}

// InputClassificationData holds the data for input classification template
type InputClassificationData struct {
	Scene                 *scene.Scene
	PlayerInput           string
	ActiveChallengeSkills []string // Pending task skills from an active challenge (e.g., ["Notice", "Stealth"])
}

// SceneResponseData holds the data for scene response template
type SceneResponseData struct {
	Scene               *scene.Scene
	CharacterContext    string
	AspectsContext      string
	ConversationContext string
	PlayerInput         string
	InteractionType     string
	OtherCharacters     []*character.Character
	TakenOutCharacters  []*character.Character // Characters defeated earlier in this scene
	ScenePurpose        string                 // Dramatic question driving this scene
}

// ConflictResponseData holds the data for conflict response template
type ConflictResponseData struct {
	Scene                *scene.Scene
	CharacterContext     string
	AspectsContext       string
	ConversationContext  string
	PlayerInput          string
	OtherCharacters      []*character.Character
	TakenOutCharacters   []*character.Character // Characters defeated earlier in this scene
	CurrentCharacterName string
	ParticipantMap       map[string]*scene.ConflictParticipant
	CharacterMap         map[string]*character.Character
	ScenePurpose         string // Dramatic question driving this scene
}

// ActionNarrativeData holds the data for action narrative template
type ActionNarrativeData struct {
	Scene                *scene.Scene
	CharacterContext     string
	AspectsContext       string
	ConversationContext  string
	Action               *action.Action
	OtherCharacters      []*character.Character
	ChallengeDescription string // Overall challenge description, if active
	ChallengeTaskDesc    string // Specific task being resolved, if any
}

// AttackContext holds information about the attack that caused damage
type AttackContext struct {
	Skill       string // The skill used to attack (e.g., "Fight", "Shoot", "Provoke")
	Description string // The narrative description of the attack
	Shifts      int    // The shifts of damage dealt
}

// ConsequenceAspectData holds the data for consequence aspect generation template
type ConsequenceAspectData struct {
	CharacterName string
	AttackerName  string
	Severity      string
	ConflictType  string
	AttackContext
}

// TakenOutData holds the data for taken out narrative template
type TakenOutData struct {
	CharacterName       string
	AttackerName        string
	AttackerHighConcept string
	ConflictType        string
	SceneDescription    string
	AttackContext
}

// NPCAttackData contains context for NPC attack narrative generation
type NPCAttackData struct {
	ConflictType       string
	Round              int
	SceneName          string
	NPCName            string
	NPCHighConcept     string
	NPCAspects         []string
	Skill              string
	TargetName         string
	TargetHighConcept  string
	SituationAspects   []scene.SituationAspect
	OutcomeDescription string
}

// NPCActionDecisionData contains context for NPC action decision
type NPCActionDecisionData struct {
	ConflictType      string
	Round             int
	SceneName         string
	SceneDescription  string
	NPCName           string
	NPCHighConcept    string
	NPCTrouble        string
	NPCAspects        []string
	NPCSkills         map[string]int
	NPCPhysicalStress []bool
	NPCMentalStress   []bool
	Targets           []NPCTargetInfo
	SituationAspects  []scene.SituationAspect
}

// NPCTargetInfo contains information about a potential NPC target
type NPCTargetInfo struct {
	ID             string
	Name           string
	HighConcept    string
	PhysicalStress []bool
	MentalStress   []bool
}

// NPCActionDecision represents the LLM's choice for NPC action
type NPCActionDecision struct {
	ActionType  string `json:"action_type"`
	Skill       string `json:"skill"`
	TargetID    string `json:"target_id"`
	Description string `json:"description"`
}

// SceneGenerationData holds the data for scene generation template
type SceneGenerationData struct {
	TransitionHint    string          // Hint from previous scene transition
	Scenario          *scene.Scenario // The current scenario context (problem, questions, setting)
	PlayerName        string          // Player character name
	PlayerHighConcept string          // Player high concept
	PlayerTrouble     string          // Player trouble aspect
	PlayerAspects     []string        // Other player aspects
	PreviousSummaries []SceneSummary  // Summaries of recent scenes (last 3)
	Complications     []string        // Unresolved threads from previous scenes to weave in
	KnownNPCs         []NPCSummary    // Named NPCs from previous scenes that can recur
}

// GeneratedScene represents the LLM response for scene generation
type GeneratedScene struct {
	SceneName        string         `json:"scene_name"`
	Description      string         `json:"description"`
	Purpose          string         `json:"purpose"`                // Dramatic question driving the scene
	OpeningHook      string         `json:"opening_hook,omitempty"` // What interesting thing is about to happen
	SituationAspects []string       `json:"situation_aspects"`
	NPCs             []GeneratedNPC `json:"npcs"`
}

// GeneratedNPC represents an NPC generated for a new scene
type GeneratedNPC struct {
	Name        string `json:"name"`
	HighConcept string `json:"high_concept"`
	Disposition string `json:"disposition"` // friendly, neutral, hostile
}

// SceneSummary holds a structured summary of a completed scene for context continuity
type SceneSummary struct {
	SceneDescription  string       `json:"scene_description" yaml:"scene_description"`
	KeyEvents         []string     `json:"key_events" yaml:"key_events"`
	NPCsEncountered   []NPCSummary `json:"npcs_encountered" yaml:"npcs_encountered"`
	AspectsDiscovered []string     `json:"aspects_discovered" yaml:"aspects_discovered"`
	UnresolvedThreads []string     `json:"unresolved_threads" yaml:"unresolved_threads"`
	HowEnded          string       `json:"how_ended" yaml:"how_ended"`
	NarrativeProse    string       `json:"narrative_prose" yaml:"narrative_prose"`
}

// NPCSummary holds brief NPC information for scene summaries
type NPCSummary struct {
	Name     string `json:"name" yaml:"name"`
	Attitude string `json:"attitude" yaml:"attitude"` // friendly, hostile, neutral, defeated, etc.
}

// SceneSummaryData holds the data for scene summary generation template
type SceneSummaryData struct {
	SceneName           string
	SceneDescription    string
	SituationAspects    []string
	ConversationHistory []ConversationEntry
	NPCsInScene         []NPCSummary
	TakenOutChars       []string
	HowEnded            string // "transition", "quit", "player_taken_out"
	TransitionHint      string
}

// ScenarioGenerationData holds the data for scenario generation template
type ScenarioGenerationData struct {
	PlayerName        string
	PlayerHighConcept string
	PlayerTrouble     string
	PlayerAspects     []string
	Genre             string // Optional genre hint
	Theme             string // Optional theme hint
}

// ScenarioResolutionData holds the data for scenario resolution check template
type ScenarioResolutionData struct {
	Scenario       *scene.Scenario
	SceneSummaries []SceneSummary
	LatestSummary  *SceneSummary
	PlayerName     string
	PlayerAspects  []string
}

// ScenarioResolutionResult represents the LLM response for resolution check
type ScenarioResolutionResult struct {
	IsResolved        bool     `json:"is_resolved"`
	AnsweredQuestions []string `json:"answered_questions"` // Questions that have been answered
	Reasoning         string   `json:"reasoning"`          // Brief explanation
}

// RecoveryNarrativeData holds the data for between-scene recovery narrative template
type RecoveryNarrativeData struct {
	CharacterName string
	SceneSetting  string
	Consequences  []RecoveryAttempt
}

// RecoveryAttempt holds fields for a single consequence recovery attempt
type RecoveryAttempt struct {
	Severity   string
	Aspect     string
	Difficulty string
	Skill      string
	RollResult int
	Outcome    string // "success" or "failure"
}

// ConversationEntry is a type alias for the canonical definition in uicontract.
// Existing code can continue using prompt.ConversationEntry without changes.
type ConversationEntry = uicontract.ConversationEntry

// ChallengeContext provides active challenge info for prompt templates.
// When non-nil, prompts adapt to guide LLM behaviour during challenges.
type ChallengeContext struct {
	Description   string   // Overall challenge description
	PendingSkills []string // Skills of pending (unresolved) tasks
}

// ActionParseTemplateData holds the data for action parse template
type ActionParseTemplateData struct {
	Character          *character.Character
	RawInput           string
	Context            string
	Scene              interface{}
	OtherCharacters    []*character.Character
	DifficultyGuidance                   // Embedded difficulty thresholds
	ChallengeContext   *ChallengeContext // Non-nil when an active challenge is in progress
}

// FateNarrationData holds the data for the fate narration template.
// After a conflict victory, the player narrates what happens to each taken-out NPC.
// This prompt asks the LLM to parse that narration into per-NPC fates.
type FateNarrationData struct {
	SceneName        string
	SceneDescription string
	ConflictType     string // "physical" or "mental"
	TakenOutNPCs     []FateNarrationNPC
	PlayerNarration  string
}

// FateNarrationNPC holds brief info about a taken-out NPC for the fate prompt.
type FateNarrationNPC struct {
	ID          string
	Name        string
	HighConcept string
}

// NPCFateResult represents the LLM's parsed fate for a single taken-out NPC.
type NPCFateResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"` // 1-3 word fate: "killed", "unconscious", "fled"
	Permanent   bool   `json:"permanent"`   // true = permanently removed from the story
}

// FateNarrationResult represents the full LLM response for fate narration.
type FateNarrationResult struct {
	Fates     []NPCFateResult `json:"fates"`
	Narrative string          `json:"narrative"`
}

// AspectGenerationRequest holds the data for aspect generation template
type AspectGenerationRequest struct {
	Character       *character.Character `json:"character"`
	Action          *action.Action       `json:"action"`
	Outcome         *dice.Outcome        `json:"outcome"`
	Context         string               `json:"context"`                    // Scene description or situational context
	TargetType      string               `json:"target_type"`                // "character", "scene", "object", "situation"
	ExistingAspects []string             `json:"existing_aspects,omitempty"` // Aspects already in play
}
