package llm

import "strings"

// CleanJSONResponse removes markdown formatting from LLM JSON responses.
// LLMs often wrap JSON output in ```json code blocks; this extracts the raw JSON.
// When multiple JSON blocks are present, the last one is returned (the corrected response).
func CleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)

	// If there are multiple JSON blocks, take the last one (the corrected response)
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

	// Fallback: simple cleanup
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}

	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return content
}
