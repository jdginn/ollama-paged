package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/schollz/progressbar/v3"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func readFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func chunk(text string, chunkSize int) ([]string, error) {
	sentences := strings.Split(text, ".")
	chunks := make([]string, 0, len(sentences))

	var thisChunk string
	for _, sentence := range sentences {
		sentence = sentence + "."
	}

	for _, sentence := range sentences {
		if len(sentence) > chunkSize {
			if thisChunk != "" {
				chunks = append(chunks, thisChunk)
				thisChunk = ""

			}
			chunks = append(chunks, sentence[:chunkSize])
			chunks = append(chunks, sentence[chunkSize:])
			continue
		}
		if len(thisChunk)+len(sentence) > chunkSize {
			chunks = append(chunks, thisChunk)
			thisChunk = ""
			continue
		}
		thisChunk = thisChunk + sentence
	}
	chunks = append(chunks, thisChunk)
	return chunks, nil
}

type ChunkCmd struct {
	InputFile string `arg:"" name:"input" help:"input file"`
	Size      int    `help:"chunk size" default:"1024"`
}

func (c ChunkCmd) Run() error {
	text, err := readFile(c.InputFile)
	if err != nil {
		panic(err)
	}
	chunks, err := chunk(text, c.Size)
	if err != nil {
		panic(err)
	}
	for _, chunk := range chunks {
		fmt.Printf(chunk + "\n\n")
	}
	return nil
}

var filterList = []string{"Here is the corrected text:", "Here is the corrected transcription:"}

func filterIgnore(text string) string {
	for _, filter := range filterList {
		text = strings.ReplaceAll(text, filter, "")
	}
	return text
}

func trimQuotes(text string) string {
	if strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
		return strings.Trim(text, `"`)
	}
	return text
}

type PromptCmd struct {
	PromptFile string `arg:"" name:"prompt" help:"prompt file"`
	InputFile  string `arg:"" name:"input" help:"input file"`
	Size       int    `help:"chunk size" default:"1024"`
	Model      string `help:"model" default:"qwen2.5:0.5b"`
}

func (c PromptCmd) Run() error {
	text, err := readFile(c.InputFile)
	if err != nil {
		panic(err)
	}
	prompt, err := readFile(c.PromptFile)
	if err != nil {
		panic(err)
	}
	chunks, err := chunk(text, c.Size)
	if err != nil {
		panic(err)
	}
	llm, err := ollama.New(ollama.WithModel(c.Model))
	result := ""
	bar := progressbar.Default(int64(len(chunks)))
	for _, chunk := range chunks {
		bar.Add(1)
		res := ""
		if chunk != "" {
			res, err = llms.GenerateFromSinglePrompt(context.Background(), llm, prompt+chunk)
			if err != nil {
				log.Fatal(err)
			}
		}
		result = result + trimQuotes(filterIgnore(res))
	}
	fmt.Println(result)
	return nil
}

var CLI struct {
	Chunk  ChunkCmd  `cmd:"" help:"Chunk a file"`
	Prompt PromptCmd `cmd:"" help:"Process a file using ollama"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}
}
