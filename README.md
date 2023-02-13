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

  # read prompt from file and --write response back
  chatgpt convo.txt
  chatgpt convo.txt --write

  # pipe content from another program, useful for ! in vim visual mode
  cat convo.txt | chatgpt

  # inspect the predifined pretexts, which set ChatGPT's mood
  chatgpt -p list
  chatgpt -p view:<name>

  # use a pretext with any of the previous modes
  chatgpt -p optimistic -i
  chatgpt -p cynic -q "Is the world going to be ok?"
  chatgpt -p teacher convo.txt

	# edit mode
	chatgpt -e ...

	# code mode
	chatgpt -c ...

	# model options (https://platform.openai.com/docs/api-reference/completions/create)
	chatgpt -T 4096   # set max tokens in reponse  [0,4096]
	chatgpt -C        # clean whitespace before sending
	chatgpt --temp    # set the temperature param  [0.0,2.0]
	chatgpt --topp    # set the TopP param         [0.0,1.0]
	chatgpt --pres    # set the Presence Penalty   [-2.0,2.0]
	chatgpt --freq    # set the Frequency Penalty  [-2.0,2.0]

Usage:
  chatgpt [file] [flags]

Flags:
  -x, --clean             remove excess whitespace from prompt before sending
  -c, --code              request code completion with ChatGPT
  -C, --count int         set the number of response options to create (default 1)
  -e, --edit              request an edit with ChatGPT
      --freq float        set the Frequency Penalty parameter
  -h, --help              help for chatgpt
  -i, --interactive       start an interactive session with ChatGPT
      --pres float        set the Presence Penalty parameter
  -p, --pretext string    pretext to add to ChatGPT input, use 'list' or 'view:<name>' to inspect predefined, '<name>' to use a pretext, or otherwise supply any custom text
  -q, --question string   ask a single question and print the response back
      --temp float        set the temperature parameter (default 1)
  -T, --tokens int        set the MaxTokens to generate per response (default 1024)
      --topp float        set the TopP parameter (default 1)
      --version           print version information
  -w, --write             write response to end of context file
```

Pretexts:

```
$ chatgpt -p list
coding
cynic
liar
optimistic
sam
teacher
thoughtful
```

Jailbreaking ChatGPT:

https://old.reddit.com/r/ChatGPT/comments/10tevu1/new_jailbreak_proudly_unveiling_the_tried_and/

Contributions:

Feel free to offer interesting pretexts or anything else
