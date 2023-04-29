package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
)

var interactiveHelp = `starting interactive session...
  'quit' to exit
  'save <filename>' to preserve
  'clear' to erase the context
  'context' to see the current context
  'prompt' to set the context to a prompt
  'tokens' to change the MaxToken param
  'count' to change number of responses
  'temp'  set the temperature param  [0.0,2.0]
  'topp'  set the TopP param         [0.0,1.0]
  'pres'  set the Presence Penalty   [-2.0,2.0]
  'freq'  set the Frequency Penalty  [-2.0,2.0]
  'model' to change the selected model
`

func RunInteractive(client *openai.Client) error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	quit := false

	// initial req setup
	var req openai.ChatCompletionRequest

	// override default model in interactive | chat mode
	if !(strings.HasPrefix(Model, "gpt-3") || strings.HasPrefix(Model, "gpt-4")) {
		Model = "gpt-3.5-turbo-0301"
		fmt.Println("using chat compatible model:", Model, "\n")
	}
	fmt.Println(interactiveHelp)
	fmt.Println(PromptText + "\n")

	req.Model = Model
	req.N = Count
	req.MaxTokens = MaxTokens
	req.Temperature = float32(Temp)
	req.TopP = float32(TopP)
	req.PresencePenalty = float32(PresencePenalty)
	req.FrequencyPenalty = float32(FrequencyPenalty)

	if PromptText != "" {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role: "system",
			Content: PromptText,
		})
	}

	// interactive loop
	for !quit {
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		question := scanner.Text()
		parts := strings.Fields(question)

		// look for commands
		switch parts[0] {
		case "quit", "q", "exit":
			quit = true
			continue

		case "clear":
			req.Messages = make([]openai.ChatCompletionMessage,0)

		case "context":
			fmt.Println("\n===== Current Context =====")
			fmt.Println(PromptText)
			fmt.Println("===========================\n")

		case "prompt":
			if len(parts) < 2 {
				fmt.Println("prompt requires an argument [list, view:<prompt>, <prompt>, <custom...>]")
				continue
			}
			p, err := handlePrompt(parts[1])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			// prime prompt with custom pretext
			fmt.Printf("setting prompt to:\n%s", p)
			PromptText = p 
			if PromptText != "" {
				msg := openai.ChatCompletionMessage{
					Role: "system",
					Content: PromptText,
				}
				// new first message or replace
				if len(req.Messages) == 0 {
					req.Messages = append(req.Messages, msg)
				} else {
					req.Messages[0] = msg
				}
			}

		case "save":
			name := parts[1]
			fmt.Printf("saving session to %s\n", name)

			err := os.WriteFile(name, []byte(PromptText), 0644)
			if err != nil {
				fmt.Println(err)
			}
			continue

		case "model":
			if len(parts) == 1 {
				fmt.Println("model is set to", req.Model)
				continue
			}

			req.Model = parts[1]
			fmt.Println("model is now", req.Model)
			continue

		case "tokens":
			if len(parts) == 1 {
				fmt.Println("tokens is set to", req.MaxTokens)
				continue
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println(err)
				continue
			}

			req.MaxTokens = c
			fmt.Println("tokens is now", req.MaxTokens)
			continue

		case "count":
			if len(parts) == 1 {
				fmt.Println("count is set to", req.N)
				continue
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println(err)
				continue
			}

			req.N = c
			fmt.Println("count is now", req.N)
			continue

		case "temp":
			if len(parts) == 1 {
				fmt.Println("temp is set to", req.Temperature)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				fmt.Println(err)
				continue
			}
			req.Temperature = float32(f)
			fmt.Println("temp is now", req.Temperature)

		case "topp":
			if len(parts) == 1 {
				fmt.Println("topp is set to", req.TopP)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				fmt.Println(err)
				continue
			}
			req.TopP = float32(f)
			fmt.Println("topp is now", req.TopP)

		case "pres":
			if len(parts) == 1 {
				fmt.Println("pres is set to", req.PresencePenalty)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				fmt.Println(err)
				continue
			}
			req.PresencePenalty = float32(f)
			fmt.Println("pres is now", req.PresencePenalty)

		case "freq":
			if len(parts) == 1 {
				fmt.Println("freq is set to", req.FrequencyPenalty)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				fmt.Println(err)
				continue
			}
			req.FrequencyPenalty = float32(f)
			fmt.Println("freq is now", req.FrequencyPenalty)

		default:
			var err error

			// add the question to the existing messages
			msg := openai.ChatCompletionMessage{
				Role: "user",
				Content: question,
			}
			req.Messages = append(req.Messages, msg)

			resp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				return err
			}
			R := resp.Choices

			final := ""

			if len(R) == 1 {
				final = R[0].Message.Content
			} else {
				for i, r := range R {
					final += fmt.Sprintf("[%d]: %s\n\n", i, r.Message.Content)
				}
				fmt.Println(final)
				ok := false
				pos := 0

				for !ok {
					fmt.Print("> ")

					if !scanner.Scan() {
						break
					}

					ans := scanner.Text()
					pos, err = strconv.Atoi(ans)
					if err != nil {
						fmt.Println(err)
						continue
					}
					if pos < 0 || pos >= Count {
						fmt.Println("choice must be between 0 and", Count-1)
						continue
					}
					ok = true
				}

				final = R[pos].Message.Content
			}

			// we add response to the prompt, this is how ChatGPT sessions keep context
			msg = openai.ChatCompletionMessage{
				Role: "assistant",
				Content: final,
			}
			req.Messages = append(req.Messages, msg)
			// print the latest portion of the conversation
			fmt.Println(final + "\n")
		}
	}

	return nil
}

func handlePrompt(prompt string) (string, error) {
	files, err := os.ReadDir(PromptDir)
	if err != nil {
		return "", err
	}

	// list and exit
	if prompt == "list" {
		for _, f := range files {
			fmt.Println(strings.TrimSuffix(f.Name(), ".txt"))
		}
		return "", nil
	}

	// are we in view mode?
	var viewMode bool
	if strings.HasPrefix(prompt, "view:") {
		prompt = strings.TrimPrefix(prompt, "view:")
		viewMode = true
	}

	// read prompt pretext
	var contents []byte
	// we loop so we know if we found a match or not
	found := false
	for _, f := range files {
		if strings.TrimSuffix(f.Name(), ".txt") == prompt {
			contents, err = os.ReadFile(filepath.Join(PromptDir, prompt + ".txt"))
			found = true
			break
		}
	}
	if err != nil {
		return "", err
	}

	// probably custom?
	if !found {
		fmt.Println("no predefined prompt found, using custom text")
		return prompt, nil
	}

	// print and exit or...
	// prime prompt with known pretext
	if viewMode {
		fmt.Println(string(contents))
		return "", nil
	} else {
		return string(contents), nil
	}
}

func RunOnce(client *openai.Client, filename string) error {
	ctx := context.Background()

	var R []string
	var err error

	// TODO, chat mode
	if CodeMode {
		// R, err = GetCodeResponse(client, ctx, PromptText)
	} else if EditMode {
		R, err = GetEditsResponse(client, ctx, PromptText, Question)
	} else {
		R, err = GetCompletionResponse(client, ctx, PromptText)
	}
	if err != nil {
		return err
	}

	final := ""
	if len(R) == 1 {
		final = R[0]
	} else {
		for i, r := range R {
			final += fmt.Sprintf("[%d]: %s\n\n", i, r)
		}
	}

	if filename == "" || !WriteBack {
		fmt.Println(final)
	} else {
		err = AppendToFile(filename, final)
		if err != nil {
			return err
		}
	}

	return nil
}

// AppendToFile provides a function to append data to an existing file,
// creating it if it doesn't exist
func AppendToFile(filename string, data string) error {
	// Open the file in append mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Append the data to the file
	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return file.Close()
}
