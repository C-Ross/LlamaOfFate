package prompt

import (
	"bytes"
	_ "embed"
	"fmt"
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

//go:embed templates/scene_generation_prompt.tmpl
var sceneGenerationPromptTemplate string

//go:embed templates/scene_summary_prompt.tmpl
var sceneSummaryPromptTemplate string

//go:embed templates/scenario_generation_prompt.tmpl
var scenarioGenerationPromptTemplate string

//go:embed templates/scenario_resolution_prompt.tmpl
var scenarioResolutionPromptTemplate string

//go:embed templates/recovery_narrative_prompt.tmpl
var recoveryNarrativePromptTemplate string

//go:embed templates/fate_narration_prompt.tmpl
var fateNarrationPromptTemplate string

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
	SceneGenerationPrompt        *template.Template
	SceneSummaryPrompt           *template.Template
	ScenarioGenerationPrompt     *template.Template
	ScenarioResolutionPrompt     *template.Template
	RecoveryNarrativePrompt      *template.Template
	FateNarrationPrompt          *template.Template
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

	// Parse the scene generation prompt template
	SceneGenerationPrompt, err = template.New("scene_generation").Parse(sceneGenerationPromptTemplate)
	if err != nil {
		panic("failed to parse scene generation prompt template: " + err.Error())
	}

	// Parse the scene summary prompt template
	SceneSummaryPrompt, err = template.New("scene_summary").Parse(sceneSummaryPromptTemplate)
	if err != nil {
		panic("failed to parse scene summary prompt template: " + err.Error())
	}

	// Parse the scenario generation prompt template
	ScenarioGenerationPrompt, err = template.New("scenario_generation").Parse(scenarioGenerationPromptTemplate)
	if err != nil {
		panic("failed to parse scenario generation prompt template: " + err.Error())
	}

	// Parse the scenario resolution prompt template
	ScenarioResolutionPrompt, err = template.New("scenario_resolution").Parse(scenarioResolutionPromptTemplate)
	if err != nil {
		panic("failed to parse scenario resolution prompt template: " + err.Error())
	}

	// Parse the recovery narrative prompt template
	RecoveryNarrativePrompt, err = template.New("recovery_narrative").Parse(recoveryNarrativePromptTemplate)
	if err != nil {
		panic("failed to parse recovery narrative prompt template: " + err.Error())
	}

	// Parse the fate narration prompt template
	FateNarrationPrompt, err = template.New("fate_narration").Parse(fateNarrationPromptTemplate)
	if err != nil {
		panic("failed to parse fate narration prompt template: " + err.Error())
	}
}

// executeTemplate is a helper that executes a template and returns the result as a string
func executeTemplate(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}
	return buf.String(), nil
}

// RenderInputClassification renders the input classification prompt
func RenderInputClassification(data InputClassificationData) (string, error) {
	return executeTemplate(InputClassificationPrompt, data)
}

// RenderSceneResponse renders the scene response prompt
func RenderSceneResponse(data SceneResponseData) (string, error) {
	return executeTemplate(SceneResponsePrompt, data)
}

// RenderConflictResponse renders the conflict response prompt
func RenderConflictResponse(data ConflictResponseData) (string, error) {
	return executeTemplate(ConflictResponsePrompt, data)
}

// RenderActionNarrative renders the action narrative prompt
func RenderActionNarrative(data ActionNarrativeData) (string, error) {
	return executeTemplate(ActionNarrativePrompt, data)
}

// RenderConsequenceAspect renders the consequence aspect prompt
func RenderConsequenceAspect(data ConsequenceAspectData) (string, error) {
	return executeTemplate(ConsequenceAspectPrompt, data)
}

// RenderTakenOut renders the taken out narrative prompt
func RenderTakenOut(data TakenOutData) (string, error) {
	return executeTemplate(TakenOutPrompt, data)
}

// RenderNPCActionDecision renders the NPC action decision prompt
func RenderNPCActionDecision(data NPCActionDecisionData) (string, error) {
	return executeTemplate(NPCActionDecisionPrompt, data)
}

// RenderNPCAttack renders the NPC attack narrative prompt
func RenderNPCAttack(data NPCAttackData) (string, error) {
	return executeTemplate(NPCAttackPrompt, data)
}

// RenderActionParse renders the action parse prompt
func RenderActionParse(data ActionParseTemplateData) (string, error) {
	return executeTemplate(ActionParsePrompt, data)
}

// RenderActionParseSystem renders the action parse system prompt
func RenderActionParseSystem() (string, error) {
	return executeTemplate(ActionParseSystemPrompt, nil)
}

// RenderAspectGeneration renders the aspect generation prompt
func RenderAspectGeneration(data AspectGenerationRequest) (string, error) {
	return executeTemplate(AspectGenerationPrompt, data)
}

// RenderAspectGenerationSystem renders the aspect generation system prompt
func RenderAspectGenerationSystem() (string, error) {
	return executeTemplate(AspectGenerationSystemPrompt, nil)
}

// RenderSceneGeneration renders the scene generation prompt
func RenderSceneGeneration(data SceneGenerationData) (string, error) {
	return executeTemplate(SceneGenerationPrompt, data)
}

// RenderSceneSummary renders the scene summary prompt
func RenderSceneSummary(data SceneSummaryData) (string, error) {
	return executeTemplate(SceneSummaryPrompt, data)
}

// RenderScenarioGeneration renders the scenario generation prompt
func RenderScenarioGeneration(data ScenarioGenerationData) (string, error) {
	return executeTemplate(ScenarioGenerationPrompt, data)
}

// RenderScenarioResolution renders the scenario resolution prompt
func RenderScenarioResolution(data ScenarioResolutionData) (string, error) {
	return executeTemplate(ScenarioResolutionPrompt, data)
}

// RenderRecoveryNarrative renders the between-scene recovery narrative prompt
func RenderRecoveryNarrative(data RecoveryNarrativeData) (string, error) {
	return executeTemplate(RecoveryNarrativePrompt, data)
}

// RenderFateNarration renders the post-victory fate narration prompt
func RenderFateNarration(data FateNarrationData) (string, error) {
	return executeTemplate(FateNarrationPrompt, data)
}
