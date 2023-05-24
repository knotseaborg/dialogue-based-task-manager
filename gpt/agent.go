package gpt

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Spake struct {
	Source  string
	Content string
}

func Complete(dialogue *Spake) {

	NextSourceSelector(dialogue)

	c := openai.NewClient(os.Getenv("DBTM_OPEN_AI_KEY"))
	rawIntroduction, err := os.ReadFile("./gpt/prompt.txt")
	if err != nil {
		log.Print(err)
	}

	ctx := context.Background()
	req := openai.CompletionRequest{
		Model:       openai.GPT3TextDavinci003,
		MaxTokens:   500,
		Prompt:      fmt.Sprintf("%s\n%s: %s", string(rawIntroduction), dialogue.Source, dialogue.Content),
		Stop:        []string{"User:", "Emma:", "API Response:"},
		Temperature: 0.3,
	}
	resp, err := c.CreateCompletion(ctx, req)
	if err != nil {
		fmt.Printf("Completion error: %v\n", err)
		return
	}
	if strings.Contains(resp.Choices[0].Text, "::") {
		splitDirectives := strings.Split(resp.Choices[0].Text, "::")
		for _, directive := range splitDirectives {
			fmt.Println(directive)
		}
	}
	//fmt.Println(string(rawIntroduction), "Hello")
	fmt.Println(resp.Choices[0].Text)
}

func NextSourceSelector(dialogue *Spake) {
	if dialogue.Source == "User" {
		dialogue.Content = dialogue.Content + "\nEmma:"
	}
}
