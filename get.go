package main

import (
	"context"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func GetCompletionResponse(client *openai.Client, ctx context.Context, question string) ([]string, error) {
	if CleanPrompt {
		question = strings.ReplaceAll(question, "\n", " ")
		question = strings.ReplaceAll(question, "  ", " ")
	}
	// insert newline at end to prevent completion of question
	if !strings.HasSuffix(question, "\n") {
		question += "\n"
	}

	req := openai.CompletionRequest{
		Model:            Model,
		MaxTokens:        MaxTokens,
		Prompt:           question,
		Echo:             Echo,
		N:                Count,
		Temperature:      float32(Temp),
		TopP:             float32(TopP),
		PresencePenalty:  float32(PresencePenalty),
		FrequencyPenalty: float32(FrequencyPenalty),
	}
	resp, err := client.CreateCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	var r []string
	for _, c := range resp.Choices {
		r = append(r, c.Text)
	}
	return r, nil
}

func GetEditsResponse(client *openai.Client, ctx context.Context, input, instruction string) ([]string, error) {
	if CleanPrompt {
		input = strings.ReplaceAll(input, "\n", " ")
		input = strings.ReplaceAll(input, "  ", " ")
	}

	m := Model
	req := openai.EditsRequest{
		Model:       &m,
		Input:       input,
		Instruction: instruction,
		N:           Count,
		Temperature: float32(Temp),
		TopP:        float32(TopP),
	}
	resp, err := client.Edits(ctx, req)
	if err != nil {
		return nil, err
	}

	var r []string
	for _, c := range resp.Choices {
		r = append(r, c.Text)
	}
	return r, nil
}


