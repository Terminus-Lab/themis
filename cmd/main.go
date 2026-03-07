package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	_ = godotenv.Load()

	openAiKey := os.Getenv("OPEN_AI_KEY")
	if openAiKey == "" {
		log.Fatal("OPEN_AI_KEY is not set") // ✅ Fatal instead of just printing
	}

	client := openai.NewClient(option.WithAPIKey(openAiKey))

	resp, err := client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4oMini, // ✅ Wrapped in openai.F()
			Messages: []openai.ChatCompletionMessageParamUnion{ // ✅ Correct type
				openai.UserMessage("Hello, how are you?"), // ✅ Helper constructor
			},
		},
	)

	if err != nil {
		log.Fatalf("ChatCompletion error: %v", err) // ✅ log.Fatalf instead of fmt.Println
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
