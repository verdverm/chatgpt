package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	gpt3 "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var LongHelp = `
Chat with ChatGPT in console.

Examples:
  # start an interactive session
  chatgpt -i

  # ask chatgpt for a one-time response
  chatgpt -q "answer me this ChatGPT..."

  # provide context to a question or conversation
  chatgpt context.txt -i
  chatgpt context.txt -q "answer me this ChatGPT..."

  # read prompt from file and --write response back
  chatgpt convo.txt
  chatgpt convo.txt --write

  # pipe content from another program, useful for ! in vim visual mode
  cat convo.txt | chatgpt

  # inspect the predifined prompts, which set ChatGPT's mood
  chatgpt -p list
  chatgpt -p view:<name>

  # use a prompts with any of the previous modes
  chatgpt -p optimistic -i
  chatgpt -p cynic -q "Is the world going to be ok?"
  chatgpt -p teacher convo.txt

	# set the directory for custom prompts
  chatgpt -P prompts -p my-prompt -i

  # edit mode
  chatgpt -e ...

  # code mode
  chatgpt -c ...

  # model options (https://platform.openai.com/docs/api-reference/completions/create)
  chatgpt -T 4096    # set max tokens in reponse  [0,4096]
  chatgpt -C         # clean whitespace before sending
  chatgpt -E         # echo back the prompt, useful for vim coding
  chatgpt --temp     # set the temperature param  [0.0,2.0]
  chatgpt --topp     # set the TopP param         [0.0,1.0]
  chatgpt --pres     # set the Presence Penalty   [-2.0,2.0]
  chatgpt --freq     # set the Frequency Penalty  [-2.0,2.0]

  # change model selection, available models are listed here:
  # https://pkg.go.dev/github.com/sashabaranov/go-openai#Client.ListModels
  chatgpt -m text-davinci-003  # set the model to text-davinci-003 (the default)
  chatgpt -m text-ada-001      # set the model to text-ada-001

`

var interactiveHelp = `starting interactive session...
  'quit' to exit
  'save <filename>' to preserve
  'tokens' to change the MaxToken param
  'count' to change number of responses
  'temp'  set the temperature param  [0.0,2.0]
  'topp'  set the TopP param         [0.0,1.0]
  'pres'  set the Presence Penalty   [-2.0,2.0]
  'freq'  set the Frequency Penalty  [-2.0,2.0]
  'model' to change the selected model
`

//go:embed prompts/*
var predefined embed.FS

var Version bool

// prompt vars
var Question string
var Prompt string
var PromptDir string
var PromptMode bool
var EditMode bool
var CodeMode bool
var CleanPrompt bool
var WriteBack bool
var PromptText string

// chatgpt vars
var MaxTokens int
var Count int
var Echo bool
var Temp float64
var TopP float64
var PresencePenalty float64
var FrequencyPenalty float64
var Model string

// internal vars
func init() {
}



/*
func GetChatCompletionResponse(client *gpt3.Client, ctx context.Context, question string) ([]string, error) {
	if CleanPrompt {
		question = strings.ReplaceAll(question, "\n", " ")
		question = strings.ReplaceAll(question, "  ", " ")
	}
	// insert newline at end to prevent completion of question
	if !strings.HasSuffix(question, "\n") {
		question += "\n"
	}

	req := gpt3.ChatCompletionRequest{
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
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	var r []string
	for _, c := range resp.Choices {
		r = append(r, c.Text)
	}
	return r, nil
}
*/

func GetCompletionResponse(client *gpt3.Client, ctx context.Context, question string) ([]string, error) {
	if CleanPrompt {
		question = strings.ReplaceAll(question, "\n", " ")
		question = strings.ReplaceAll(question, "  ", " ")
	}
	// insert newline at end to prevent completion of question
	if !strings.HasSuffix(question, "\n") {
		question += "\n"
	}

	req := gpt3.CompletionRequest{
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

func GetEditsResponse(client *gpt3.Client, ctx context.Context, input, instruction string) ([]string, error) {
	if CleanPrompt {
		input = strings.ReplaceAll(input, "\n", " ")
		input = strings.ReplaceAll(input, "  ", " ")
	}

	m := Model
	req := gpt3.EditsRequest{
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

func GetCodeResponse(client *gpt3.Client, ctx context.Context, question string) ([]string, error) {
	if CleanPrompt {
		question = strings.ReplaceAll(question, "\n", " ")
		question = strings.ReplaceAll(question, "  ", " ")
	}
	// insert newline at end to prevent completion of question
	if !strings.HasSuffix(question, "\n") {
		question += "\n"
	}

	req := gpt3.CompletionRequest{
		Model:            gpt3.CodexCodeDavinci002,
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

func printVersion() {
	info, _ := debug.ReadBuildInfo()
	GoVersion := info.GoVersion
	Commit := ""
	BuildDate := ""
	dirty := false

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			Commit = s.Value
		}
		if s.Key == "vcs.time" {
			BuildDate = s.Value
		}
		if s.Key == "vcs.modified" {
			if s.Value == "true" {
				dirty = true
			}
		}
	}
	if dirty {
		Commit += "+dirty"
	}

	fmt.Printf("%s %s %s\n", Commit, BuildDate, GoVersion)
}

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

func main() {

	apiKey := os.Getenv("CHATGPT_API_KEY")
	if apiKey == "" {
		fmt.Println("CHATGPT_API_KEY environment var is missing\nVisit https://platform.openai.com/account/api-keys to get one\n")
		os.Exit(1)
	}

	if PromptDir == "" {
		if v := os.Getenv("CHATGPT_PROMPT_DIR"); v != "" {
			PromptDir = v
		}
	}

	client := gpt3.NewClient(apiKey)

	rootCmd := &cobra.Command{
		Use:   "chatgpt [file]",
		Short: "Chat with ChatGPT in console.",
		Long:  LongHelp,
		Run: func(cmd *cobra.Command, args []string) {
			if Version {
				printVersion()
				os.Exit(0)
			}

			var err error
			var filename string

			// We build up PromptText as we go, based on flags

			// Handle the prompt flag
			if Prompt != "" {
				var files []fs.DirEntry

				if PromptDir == "" {
					files, err = predefined.ReadDir("prompts")
					if err != nil {
						panic(err)
					}
				} else {
					files, err = os.ReadDir(PromptDir)
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}

				// list and exit
				if Prompt == "list" {
					for _, f := range files {
						fmt.Println(strings.TrimSuffix(f.Name(), ".txt"))
					}
					os.Exit(0)
				}

				// are we in view mode?
				var viewMode bool
				if strings.HasPrefix(Prompt, "view:") {
					Prompt = strings.TrimPrefix(Prompt, "view:")
					viewMode = true
				}

				// read prompt pretext
				var contents []byte
				if PromptDir == "" {
					contents, err = predefined.ReadFile("prompts/" + Prompt + ".txt")
				} else {
					contents, err = os.ReadFile(filepath.Join(PromptDir, Prompt + ".txt"))
				}
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				// print and exit or...
				// prime prompt with known pretext
				if viewMode {
					fmt.Println(string(contents))
					os.Exit(0)
				} else {
					PromptText = string(contents)
				}

				// prime prompt with custom pretext
				if PromptText == "" {
					PromptText = Prompt
				}

			}

			// no args, or interactive... read from stdin
			// this is mainly for replacing text in vim
			if len(args) == 0 && !PromptMode {
				reader := bufio.NewReader(os.Stdin)
				var buf bytes.Buffer
				for {
					b, err := reader.ReadByte()
					if err != nil {
						break
					}
					buf.WriteByte(b)
				}
				PromptText += buf.String()
			} else if len(args) == 1 {
				// if we have an arg, add it to the prompt
				filename = args[0]
				content, err := os.ReadFile(filename)
				if err != nil {
					fmt.Println(err)
					return
				}
				PromptText += string(content)
			}

			// if there is a question, it comes last in the prompt
			if Question != "" && !EditMode {
				PromptText += "\n" + Question
			}

			// interactive or file mode
			if PromptMode {
				fmt.Println(interactiveHelp)
				fmt.Println(PromptText)
				err = RunPrompt(client)
			} else {
				// empty filename (no args) prints to stdout
				err = RunOnce(client, filename)
			}

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

		},
	}

	// setup flags
	rootCmd.Flags().BoolVarP(&Version, "version", "", false, "print version information")

	// prompt releated
	rootCmd.Flags().StringVarP(&Question, "question", "q", "", "ask a single question and print the response back")
	rootCmd.Flags().StringVarP(&Prompt, "prompt", "p", "", "prompt to add to ChatGPT input, use 'list' or 'view:<name>' to inspect predefined, '<name>' to use a prompt, or otherwise supply any custom text")
	rootCmd.Flags().StringVarP(&PromptDir, "prompt-dir", "P", "", "directory containing custom prompts, if not set the embedded defaults are used")
	rootCmd.Flags().BoolVarP(&PromptMode, "interactive", "i", false, "start an interactive session with ChatGPT")
	rootCmd.Flags().BoolVarP(&EditMode, "edit", "e", false, "request an edit with ChatGPT")
	rootCmd.Flags().BoolVarP(&CodeMode, "code", "c", false, "request code completion with ChatGPT")
	rootCmd.Flags().BoolVarP(&CleanPrompt, "clean", "x", false, "remove excess whitespace from prompt before sending")
	rootCmd.Flags().BoolVarP(&WriteBack, "write", "w", false, "write response to end of context file")

	// params related
	rootCmd.Flags().IntVarP(&MaxTokens, "tokens", "T", 1024, "set the MaxTokens to generate per response")
	rootCmd.Flags().IntVarP(&Count, "count", "C", 1, "set the number of response options to create")
	rootCmd.Flags().BoolVarP(&Echo, "echo", "E", false, "Echo back the prompt, useful for vim coding")
	rootCmd.Flags().Float64VarP(&Temp, "temp", "", 0.7, "set the temperature parameter")
	rootCmd.Flags().Float64VarP(&TopP, "topp", "", 1.0, "set the TopP parameter")
	rootCmd.Flags().Float64VarP(&PresencePenalty, "pres", "", 0.0, "set the Presence Penalty parameter")
	rootCmd.Flags().Float64VarP(&FrequencyPenalty, "freq", "", 0.0, "set the Frequency Penalty parameter")
	rootCmd.Flags().StringVarP(&Model, "model", "m", gpt3.GPT3TextDavinci003, "select the model to use with -q or -e")

	// run the command
	rootCmd.Execute()
}

func RunPrompt(client *gpt3.Client) error {
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

			if CodeMode {
				R, err = GetCodeResponse(client, ctx, PromptText)
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

func RunOnce(client *gpt3.Client, filename string) error {
	ctx := context.Background()

	var R []string
	var err error

	if CodeMode {
		R, err = GetCodeResponse(client, ctx, PromptText)
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
