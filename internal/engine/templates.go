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

// Template instances
var (
	AspectGenerationPrompt       *template.Template
	AspectGenerationSystemPrompt *template.Template
	ActionParseSystemPrompt      *template.Template
	ActionParsePrompt            *template.Template
	InputClassificationPrompt    *template.Template
	SceneResponsePrompt          *template.Template
	ActionNarrativePrompt        *template.Template
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
}
