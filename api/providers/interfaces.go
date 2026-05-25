package providers

//go:generate mockgen -source=interfaces.go -destination=../services/mocks/mock_providers.go -package=mocks

import "context"

// PushProviderIface abstracts ExpoPushProvider for testing.
type PushProviderIface interface {
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}
