package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var LongHelp = `
Work with ChatGPT in console.

Examples:
  # start an interactive session with gpt-3.5 or gpt-4
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

	# set the directory for custom prompts, defaults to "prompts"
  chatgpt -P custom-prompts -p my-prompt -i

  # inspect the predifined prompts, which set ChatGPT's mood
  chatgpt -p list
  chatgpt -p view:<name>

  # use a prompts with any of the previous modes
  chatgpt -p optimistic -i
  chatgpt -p cynic -q "Is the world going to be ok?"
  chatgpt -p teacher convo.txt

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

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment var is missing\nVisit https://platform.openai.com/account/api-keys to get one\n")
		os.Exit(1)
	}

	if PromptDir == "" {
		if v := os.Getenv("CHATGPT_PROMPT_DIR"); v != "" {
			PromptDir = v
		}
	}

	client := openai.NewClient(apiKey)

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
				p, err := handlePrompt(Prompt)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if p == "" {
					os.Exit(0)
				}

				// prime prompt with custom pretext
				PromptText = p 
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
				err = RunInteractive(client)
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
	rootCmd.Flags().StringVarP(&PromptDir, "prompt-dir", "P", "prompts", "directory containing custom prompts, if not set the embedded defaults are used")
	rootCmd.Flags().BoolVarP(&PromptMode, "interactive", "i", false, "start an interactive session with ChatGPT")
	rootCmd.Flags().BoolVarP(&EditMode, "edit", "e", false, "request an edit with ChatGPT")
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
	rootCmd.Flags().StringVarP(&Model, "model", "m", openai.GPT3TextDavinci003, "select the model to use with -q or -e")

	// run the command
	rootCmd.Execute()
}
