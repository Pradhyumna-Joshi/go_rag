package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Pradhyumna-Joshi/go_rag/components"
	"github.com/Pradhyumna-Joshi/go_rag/internal/rag"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

type Server struct {
	config Config
}

type Config struct {
	Addr string
}

func NewServer(config Config) *Server {
	return &Server{config}
}

var upgrader = websocket.Upgrader{}

func (s *Server) mount() http.Handler {

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		components.Home().Render(r.Context(), w)
	})

	embedderllm, err := ollama.New(ollama.WithModel("nomic-embed-text-v2-moe"))
	if err != nil {
		log.Fatal(err)
	}

	embedder, err := embeddings.NewEmbedder(embedderllm)
	if err != nil {
		log.Fatal(err)
	}

	store, err := chroma.New(
		chroma.WithChromaURL("http://localhost:8000"),
		chroma.WithEmbedder(embedder),
		chroma.WithDistanceFunction("cosine"),
		chroma.WithNameSpace(uuid.New().String()),
	)

	// generate embeeddings
	mux.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rag.GenerateEmbeddings(string(content), &store)

		json.NewEncoder(w).Encode("Success")
	})

	// user query
	/*
		mux.HandleFunc("POST /query", func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming not supported!!", http.StatusInternalServerError)
				return
			}

			query := r.FormValue("query")
			log.Println(query)
			// call llm with query and context
			_ = rag.Call(query, &store, func(b []byte) error {
				fmt.Fprintf(w, `<span hx-swap-oob="beforeend:#response">%s</span>`, b)
				flusher.Flush()
				return nil
			})

			//components.ResponseItem(resp).Render(r.Context(), w)

		})

	*/

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		for {

			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Fatal(err)
			}
			rag.Call(string(msg), &store, func(b []byte) error {
				conn.WriteMessage(websocket.TextMessage, b)
				return nil
			})
		}

	})

	return mux
}

func (s *Server) run(h http.Handler) error {
	server := http.Server{
		Addr:         s.config.Addr,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	log.Printf("Server running on port %s", s.config.Addr)
	return server.ListenAndServe()
}
