package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// OllamaEmbedder calls a self-hosted Ollama /api/embed endpoint that sits behind a bearer-token proxy
// (EMBEDDING_URL + EMBEDDING_TOKEN), e.g. https://embed.weworship.id serving bge-m3.
type OllamaEmbedder struct {
	baseURL string
	token   string
	model   string
	client  *http.Client
}

// NewEmbedderFromEnv returns nil (semantic search disabled) when EMBEDDING_URL or EMBEDDING_TOKEN is unset.
// EMBEDDING_MODEL defaults to bge-m3, the model the song_embeddings table (vector(1024)) is sized for.
func NewEmbedderFromEnv() EmbedderIface {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("EMBEDDING_URL")), "/")
	token := strings.TrimSpace(os.Getenv("EMBEDDING_TOKEN"))
	if baseURL == "" || token == "" {
		log.Printf("[embeddings] EMBEDDING_URL/EMBEDDING_TOKEN not set — semantic search disabled, using text search only")
		return nil
	}
	model := strings.TrimSpace(os.Getenv("EMBEDDING_MODEL"))
	if model == "" {
		model = "bge-m3"
	}
	return &OllamaEmbedder{
		baseURL: baseURL,
		token:   token,
		model:   model,
		// Callers pass their own deadline via ctx; this is only a ceiling for batch indexing.
		client: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (e *OllamaEmbedder) Model() string { return e.model }

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed returns one vector per input, in input order.
func (e *OllamaEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(embedRequest{Model: e.model, Input: inputs})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)

	res, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return nil, fmt.Errorf("embedding service returned %d: %s", res.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var out embedResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(out.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("embedding service returned %d vectors for %d inputs", len(out.Embeddings), len(inputs))
	}
	return out.Embeddings, nil
}
