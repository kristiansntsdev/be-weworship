package providers

import (
	"context"
	"fmt"
	"log"
	"os"

	fcm "github.com/tevjef/go-fcm"
)

// FCMProvider wraps the go-fcm client and exposes helper methods for sending
// push notifications via the Firebase Cloud Messaging HTTP v1 API.
type FCMProvider struct {
	client    *fcm.Client
	projectID string
	// tempFile holds the path of a credentials file written from FCM_CREDENTIALS_JSON.
	// It is cleaned up when Close() is called.
	tempFile string
}

// NewFCMProvider creates a new FCMProvider.
//
// It resolves credentials using the following priority:
//  1. FCM_CREDENTIALS_JSON env var – raw JSON content (recommended for production/Vercel).
//     The content is written to a temp file so go-fcm can read it.
//  2. credentialsPath argument – path to a local service-account JSON file (local dev).
func NewFCMProvider(projectID, credentialsPath string) (*FCMProvider, error) {
	if projectID == "" {
		return nil, fmt.Errorf("fcm: FCM_PROJECT_ID is required")
	}

	resolvedPath := credentialsPath
	var tempFile string

	// Priority 1 – JSON content supplied directly via env var (production).
	if jsonContent := os.Getenv("FCM_CREDENTIALS_JSON"); jsonContent != "" {
		f, err := os.CreateTemp("", "fcm-sa-*.json")
		if err != nil {
			return nil, fmt.Errorf("fcm: failed to create temp credentials file: %w", err)
		}
		if _, err := f.WriteString(jsonContent); err != nil {
			f.Close()
			os.Remove(f.Name())
			return nil, fmt.Errorf("fcm: failed to write temp credentials file: %w", err)
		}
		f.Close()
		resolvedPath = f.Name()
		tempFile = f.Name()
		log.Printf("[fcm] using credentials from FCM_CREDENTIALS_JSON env var")
	}

	if resolvedPath == "" {
		return nil, fmt.Errorf("fcm: credentials not found – set FCM_CREDENTIALS_PATH (local) or FCM_CREDENTIALS_JSON (production)")
	}

	client, err := fcm.NewClient(projectID, resolvedPath)
	if err != nil {
		if tempFile != "" {
			os.Remove(tempFile)
		}
		return nil, fmt.Errorf("fcm: failed to create client: %w", err)
	}

	log.Printf("[fcm] provider initialised (project: %s)", projectID)
	return &FCMProvider{client: client, projectID: projectID, tempFile: tempFile}, nil
}

// Close removes any temporary credentials file created from FCM_CREDENTIALS_JSON.
// Call this when shutting down the application.
func (p *FCMProvider) Close() {
	if p.tempFile != "" {
		os.Remove(p.tempFile)
	}
}

// ProjectID returns the Firebase project ID this provider is configured for.
func (p *FCMProvider) ProjectID() string {
	return p.projectID
}

// defaultApnsPayload returns the APNS payload map required by ApnsConfig.Payload.
// ApnsConfig.Payload is map[string]interface{}, so we use ApnsPayload.MustToMap()
// to convert the typed struct.
func defaultApnsPayload() map[string]interface{} {
	return (&fcm.ApnsPayload{
		Aps: &fcm.ApsDictionary{
			Badge:            1,
			ContentAvailable: int(fcm.ApnsContentAvailable),
		},
	}).MustToMap()
}

// SendToToken sends a push notification to a single device identified by its
// FCM registration token.
//
//	token – the device registration token
//	title – notification title
//	body  – notification body
//	data  – optional key-value pairs delivered as a data payload (may be nil)
func (p *FCMProvider) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	msg := &fcm.SendRequest{
		Message: &fcm.Message{
			Token: token,
			Notification: &fcm.Notification{
				Title: title,
				Body:  body,
			},
			Android: &fcm.AndroidConfig{
				Priority: string(fcm.AndroidHighPriority),
				Notification: &fcm.AndroidNotification{
					Icon:        "ic_notification",
					ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				},
			},
			Apns: &fcm.ApnsConfig{
				Payload: defaultApnsPayload(),
			},
			Data: data,
		},
	}

	resp, err := p.client.Send(msg)
	if err != nil {
		return fmt.Errorf("fcm: send to token failed: %w", err)
	}

	log.Printf("[fcm] SendToToken OK token=%.20s... name=%s", token, resp.Name)
	return nil
}

// SendToTopic sends a push notification to all devices subscribed to the given
// FCM topic.
//
//	topic – topic name (without the /topics/ prefix)
//	title – notification title
//	body  – notification body
//	data  – optional key-value pairs delivered as a data payload (may be nil)
func (p *FCMProvider) SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) error {
	msg := &fcm.SendRequest{
		Message: &fcm.Message{
			Topic: topic,
			Notification: &fcm.Notification{
				Title: title,
				Body:  body,
			},
			Android: &fcm.AndroidConfig{
				Priority: string(fcm.AndroidHighPriority),
				Notification: &fcm.AndroidNotification{
					Icon:        "ic_notification",
					ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				},
			},
			Apns: &fcm.ApnsConfig{
				Payload: defaultApnsPayload(),
			},
			Data: data,
		},
	}

	resp, err := p.client.Send(msg)
	if err != nil {
		return fmt.Errorf("fcm: send to topic failed: %w", err)
	}

	log.Printf("[fcm] SendToTopic OK topic=%s name=%s", topic, resp.Name)
	return nil
}

// SendValidateOnly sends the message in validate-only mode – FCM validates the
// request but does NOT deliver the notification. Useful for testing credentials
// and message structure without real delivery.
func (p *FCMProvider) SendValidateOnly(ctx context.Context, token, title, body string) error {
	msg := &fcm.SendRequest{
		ValidateOnly: true,
		Message: &fcm.Message{
			Token: token,
			Notification: &fcm.Notification{
				Title: title,
				Body:  body,
			},
		},
	}

	resp, err := p.client.Send(msg)
	if err != nil {
		return fmt.Errorf("fcm: validate-only send failed: %w", err)
	}

	log.Printf("[fcm] validate-only response: %+v", resp)
	return nil
}
