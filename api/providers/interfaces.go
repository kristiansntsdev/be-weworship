package providers

//go:generate mockgen -source=interfaces.go -destination=../services/mocks/mock_providers.go -package=mocks

import "context"

// PushProviderIface abstracts ExpoPushProvider for testing.
type PushProviderIface interface {
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

// EmbedderIface abstracts the text-embedding service used for semantic song search.
type EmbedderIface interface {
	Model() string
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}
