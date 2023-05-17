package gpt

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

func Transcript() (string, error) {
	c := openai.NewClient(os.Getenv("DBTM_OPEN_AI_KEY"))
	ctx := context.Background()

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: os.Getenv("DBTM_AUDIO_INPUT_PATH"),
	}
	resp, err := c.CreateTranscription(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
