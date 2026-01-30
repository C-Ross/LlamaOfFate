package engine

import (
	_ "embed"
	"text/template"
)

// Embedded template files
//
//go:embed templates/aspect_generation_prompt.tmpl
var aspectGenerationPromptTemplate string

//go:embed templates/aspect_generation_system_prompt.tmpl
var aspectGenerationSystemPromptTemplate string

//go:embed templates/action_parse_system_prompt.tmpl
var actionParseSystemPromptTemplate string

//go:embed templates/action_parse_prompt.tmpl
var actionParsePromptTemplate string

//go:embed templates/input_classification_prompt.tmpl
var inputClassificationPromptTemplate string

//go:embed templates/scene_response_prompt.tmpl
var sceneResponsePromptTemplate string

//go:embed templates/action_narrative_prompt.tmpl
var actionNarrativePromptTemplate string

//go:embed templates/conflict_response_prompt.tmpl
var conflictResponsePromptTemplate string

//go:embed templates/npc_attack_prompt.tmpl
var npcAttackPromptTemplate string

//go:embed templates/npc_action_decision_prompt.tmpl
var npcActionDecisionPromptTemplate string

//go:embed templates/consequence_aspect_prompt.tmpl
var consequenceAspectPromptTemplate string

//go:embed templates/taken_out_prompt.tmpl
var takenOutPromptTemplate string

// Template instances
var (
	AspectGenerationPrompt       *template.Template
	AspectGenerationSystemPrompt *template.Template
	ActionParseSystemPrompt      *template.Template
	ActionParsePrompt            *template.Template
	InputClassificationPrompt    *template.Template
	SceneResponsePrompt          *template.Template
	ActionNarrativePrompt        *template.Template
	ConflictResponsePrompt       *template.Template
	NPCAttackPrompt              *template.Template
	NPCActionDecisionPrompt      *template.Template
	ConsequenceAspectPrompt      *template.Template
	TakenOutPrompt               *template.Template
)

func init() {
	var err error

	// Parse the aspect generation prompt template
	AspectGenerationPrompt, err = template.New("aspect_generation").Parse(aspectGenerationPromptTemplate)
	if err != nil {
		panic("failed to parse aspect generation prompt template: " + err.Error())
	}

	// Parse the aspect generation system prompt template
	AspectGenerationSystemPrompt, err = template.New("aspect_generation_system").Parse(aspectGenerationSystemPromptTemplate)
	if err != nil {
		panic("failed to parse aspect generation system prompt template: " + err.Error())
	}

	// Parse the action parse system prompt template
	ActionParseSystemPrompt, err = template.New("action_parse_system").Parse(actionParseSystemPromptTemplate)
	if err != nil {
		panic("failed to parse action parse system prompt template: " + err.Error())
	}

	// Parse the action parse prompt template
	ActionParsePrompt, err = template.New("action_parse").Parse(actionParsePromptTemplate)
	if err != nil {
		panic("failed to parse action parse prompt template: " + err.Error())
	}

	// Parse the input classification prompt template
	InputClassificationPrompt, err = template.New("input_classification").Parse(inputClassificationPromptTemplate)
	if err != nil {
		panic("failed to parse input classification prompt template: " + err.Error())
	}

	// Parse the scene response prompt template
	SceneResponsePrompt, err = template.New("scene_response").Parse(sceneResponsePromptTemplate)
	if err != nil {
		panic("failed to parse scene response prompt template: " + err.Error())
	}

	// Parse the action narrative prompt template
	ActionNarrativePrompt, err = template.New("action_narrative").Parse(actionNarrativePromptTemplate)
	if err != nil {
		panic("failed to parse action narrative prompt template: " + err.Error())
	}

	// Parse the conflict response prompt template
	ConflictResponsePrompt, err = template.New("conflict_response").Parse(conflictResponsePromptTemplate)
	if err != nil {
		panic("failed to parse conflict response prompt template: " + err.Error())
	}

	// Parse the NPC attack prompt template
	NPCAttackPrompt, err = template.New("npc_attack").Parse(npcAttackPromptTemplate)
	if err != nil {
		panic("failed to parse NPC attack prompt template: " + err.Error())
	}

	// Parse the NPC action decision prompt template
	NPCActionDecisionPrompt, err = template.New("npc_action_decision").Parse(npcActionDecisionPromptTemplate)
	if err != nil {
		panic("failed to parse NPC action decision prompt template: " + err.Error())
	}

	// Parse the consequence aspect prompt template
	ConsequenceAspectPrompt, err = template.New("consequence_aspect").Parse(consequenceAspectPromptTemplate)
	if err != nil {
		panic("failed to parse consequence aspect prompt template: " + err.Error())
	}

	// Parse the taken out prompt template
	TakenOutPrompt, err = template.New("taken_out").Parse(takenOutPromptTemplate)
	if err != nil {
		panic("failed to parse taken out prompt template: " + err.Error())
	}
}
