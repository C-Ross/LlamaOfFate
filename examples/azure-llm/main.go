package main

import (
	"context"
	"fmt"
	"log"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
)

func main() {
	// Load configuration from file
	config, err := azure.LoadConfig("configs/azure-llm.yaml")
	if err != nil {
		log.Fatalf("Failed to load Azure config: %v", err)
	}

	// Create Azure ML client
	client := azure.NewClient(*config)

	// Print model information
	modelInfo := client.GetModelInfo()
	fmt.Printf("Using model: %s (%s)\n", modelInfo.Name, modelInfo.Provider)
	fmt.Printf("Max tokens: %d\n", modelInfo.MaxTokens)
	fmt.Printf("Description: %s\n\n", modelInfo.Description)

	// Example 1: Simple chat completion
	fmt.Println("=== Example 1: Simple Chat Completion ===")
	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant for a text-based RPG game using the Fate Core system."},
			{Role: "user", Content: "What are the four basic actions in Fate Core?"},
		},
		MaxTokens:        2048,
		Temperature:      0.8,
		TopP:             0.1,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
	}

	ctx := context.Background()
	response, err := client.ChatCompletion(ctx, req)
	if err != nil {
		log.Fatalf("Chat completion failed: %v", err)
	}

	fmt.Printf("Response: %s\n", response.Choices[0].Message.Content)
	fmt.Printf("Tokens used: %d (prompt: %d, completion: %d)\n\n",
		response.Usage.TotalTokens, response.Usage.PromptTokens, response.Usage.CompletionTokens)

	// Example 2: Streaming chat completion
	fmt.Println("=== Example 2: Streaming Chat Completion ===")
	streamReq := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant for a text-based RPG game using the Fate Core system."},
			{Role: "user", Content: "Describe a magical forest scene for a Fate Core adventure, including potential aspects."},
		},
		MaxTokens:        2048,
		Temperature:      0.8,
		TopP:             0.1,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
	}

	fmt.Print("Streaming response: ")
	err = client.ChatCompletionStream(ctx, streamReq, func(chunk llm.CompletionResponse) error {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Streaming completion failed: %v", err)
	}

	fmt.Println("\n\nDone!")
}
