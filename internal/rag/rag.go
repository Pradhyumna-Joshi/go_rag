package rag

import (
	"context"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

func GenerateEmbeddings(data string, store *chroma.Store) {

	splitter := textsplitter.NewRecursiveCharacter(textsplitter.WithChunkSize(1000), textsplitter.WithChunkOverlap(100))

	documents, err := textsplitter.CreateDocuments(splitter, []string{data}, nil)
	if err != nil {
		log.Fatal(err)
	}

	_, err = store.AddDocuments(context.Background(), documents)
	if err != nil {
		log.Fatal(err)
	}

}

func Call(query string, store *chroma.Store, callback func([]byte) error) string {

	docs, err := store.SimilaritySearch(context.Background(), query, 5)
	if err != nil {
		log.Fatal(err)
	}

	contextText := ""
	for _, d := range docs {
		contextText += d.PageContent
	}

	prompt := prompts.NewPromptTemplate(
		`You are a helpful assistant.
		Answer ONLY from the provided transcript context. 
		If the transcript is insufficient, just say you don't know 
	
		Context:
		{{.context}} 
		
		Question:
		{{.question}}`,
		[]string{"context", "question"},
	)

	formattedPrompt, err := prompt.Format(map[string]any{
		"context":  contextText,
		"question": query,
	})
	if err != nil {
		log.Fatal(err)
	}

	llm, err := ollama.New(ollama.WithModel("mistral"))
	if err != nil {
		log.Fatal(err)
	}
	resp, err := llm.Call(context.Background(), formattedPrompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		return callback(chunk)
	}))
	if err != nil {
		log.Fatal(err)
	}

	return resp

}
