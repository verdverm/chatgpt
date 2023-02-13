# chatgpt

CLI application for working with ChatGPT.
Interactive or file based session with context and moods.

```
go install github.com/verdverm/chatgpt@latest

chatgpt -h
```

Authentication:

Set `CHATGPT_API_KEY`, which you can get here: https://platform.openai.com/account/api-keys

Examples:

```
Chat with ChatGPT in console.

Examples:
  # start an interactive session
  chatgpt -i

  # ask chatgpt for a one-time response
  chatgpt -q "answer me this ChatGPT..."

  # provide context to a question or conversation
  chatgpt context.txt -i
  chatgpt context.txt -q "answer me this ChatGPT..."

  # read context from file and write response back
  chatgpt convo.txt

  # pipe content from another program, useful for ! in vim visual mode
  cat convo.txt | chatgpt

  # inspect the predifined pretexts, which set ChatGPT's mood
  chatgpt -p list
  chatgpt -p view:<name>

  # use a pretext with any of the previous modes
  chatgpt -p optimistic -i
  chatgpt -p cynic -q "Is the world going to be ok?"
  chatgpt -p teacher convo.txt

	# extra options
	chatgpt -t 4096   # set max tokens in reponse
	chatgpt -c        # clean whitespace before sending

Usage:
  chatgpt [file] [flags]

Flags:
  -c, --clean             remove excess whitespace from prompt before sending
  -h, --help              help for chatgpt
  -i, --interactive       start an interactive session with ChatGPT
  -p, --pretext string    pretext to add to ChatGPT input, use 'list' or 'view:<name>' to inspect predefined, '<name>' to use a pretext, or otherwise supply any custom text
  -q, --question string   ask a single question and print the response back
  -t, --tokens int        set the MaxTokens to generate per response (default 420)
```

