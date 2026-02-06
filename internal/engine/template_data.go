package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// Template data types for LLM prompt generation.
// These structs are passed to Go templates to render prompts.

// InputClassificationData holds the data for input classification template
type InputClassificationData struct {
	Scene       *scene.Scene
	PlayerInput string
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
}

// ActionNarrativeData holds the data for action narrative template
type ActionNarrativeData struct {
	Scene               *scene.Scene
	CharacterContext    string
	AspectsContext      string
	ConversationContext string
	Action              *action.Action
	OtherCharacters     []*character.Character
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
	TransitionHint     string         // Hint from previous scene transition
	SettingContext     string         // Genre/world description
	PlayerName         string         // Player character name
	PlayerHighConcept  string         // Player high concept
	PlayerTrouble      string         // Player trouble aspect
	PlayerAspects      []string       // Other player aspects
	RecentEvents       string         // Summary of recent events (optional, deprecated - use PreviousSummaries)
	PreviousSummaries  []SceneSummary // Summaries of recent scenes (last 3)
}

// GeneratedScene represents the LLM response for scene generation
type GeneratedScene struct {
	SceneName        string         `json:"scene_name"`
	Description      string         `json:"description"`
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
