package engine

import (
	_ "embed"
	"text/template"
)

// Embedded template files
//
//go:embed templates/aspect_generation_prompt.tmpl
var aspectGenerationPromptTemplate string

//go:embed templates/system_prompt.tmpl
var systemPromptTemplate string

//go:embed templates/action_parse_system_prompt.tmpl
var actionParseSystemPromptTemplate string

//go:embed templates/action_parse_prompt.tmpl
var actionParsePromptTemplate string

// Template instances
var (
	AspectGenerationPrompt     *template.Template
	SystemPrompt              *template.Template
	ActionParseSystemPrompt   *template.Template
	ActionParsePrompt         *template.Template
)

func init() {
	var err error

	// Parse the aspect generation prompt template
	AspectGenerationPrompt, err = template.New("aspect_generation").Parse(aspectGenerationPromptTemplate)
	if err != nil {
		panic("failed to parse aspect generation prompt template: " + err.Error())
	}

	// Parse the system prompt template
	SystemPrompt, err = template.New("system_prompt").Parse(systemPromptTemplate)
	if err != nil {
		panic("failed to parse system prompt template: " + err.Error())
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
}
