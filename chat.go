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

func RunPrompt(client *openai.Client) error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	quit := false

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
			PromptText = ""

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
				fmt.Println("model is set to", Model)
				continue
			}

			Model = parts[1]
			fmt.Println("model is now", Model)
			continue

		case "tokens":
			if len(parts) == 1 {
				fmt.Println("tokens is set to", MaxTokens)
				continue
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println(err)
				continue
			}

			MaxTokens = c
			fmt.Println("tokens is now", MaxTokens)
			continue

		case "count":
			if len(parts) == 1 {
				fmt.Println("count is set to", Count)
				continue
			}
			c, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println(err)
				continue
			}

			Count = c
			fmt.Println("count is now", Count)
			continue

		case "temp":
			if len(parts) == 1 {
				fmt.Println("temp is set to", Temp)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			Temp = f
			fmt.Println("temp is now", Temp)

		case "topp":
			if len(parts) == 1 {
				fmt.Println("topp is set to", TopP)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			TopP = f
			fmt.Println("topp is now", TopP)

		case "pres":
			if len(parts) == 1 {
				fmt.Println("pres is set to", PresencePenalty)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			PresencePenalty = f
			fmt.Println("pres is now", PresencePenalty)

		case "freq":
			if len(parts) == 1 {
				fmt.Println("freq is set to", FrequencyPenalty)
				continue
			}
			f, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			FrequencyPenalty = f
			fmt.Println("freq is now", FrequencyPenalty)

		default:
			// add the question to the existing prompt text, to keep context
			PromptText += "\n> " + question
			var R []string
			var err error

			// TODO, chat mode?
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

				final = R[pos]
			}

			// we add response to the prompt, this is how ChatGPT sessions keep context
			PromptText += "\n" + strings.TrimSpace(final)
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
