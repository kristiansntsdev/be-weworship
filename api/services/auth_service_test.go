package services

import (
	"database/sql"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var testJWTSecret = []byte("test-secret-key-at-least-32-bytes!")

func newTestAuthService(repo *mocks.AuthRepo) *AuthService {
	return &AuthService{repo: repo, jwtSecret: testJWTSecret, google: GoogleConfig{ClientURL: "http://localhost:3000"}}
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestAuthService_Register_Success(t *testing.T) {
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return nil, nil // no duplicate
		},
		CreateLocalFn: func(name, email, _ string) (*models.User, error) {
			return &models.User{
				ID: 1, Username: name, Email: email,
				Role: "user", Status: "active",
				Password: sql.NullString{String: "hashed", Valid: true},
			}, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	result, status, err := svc.Register("alice_user", "alice@example.com", "password123")

	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.NotEmpty(t, result["token"])
	user, ok := result["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", user["email"])
}

func TestAuthService_Register_EmptyName(t *testing.T) {
	svc := newTestAuthService(&mocks.AuthRepo{})
	_, status, err := svc.Register("", "alice@example.com", "password123")
	assert.Equal(t, 400, status)
	assert.ErrorContains(t, err, "required")
}

func TestAuthService_Register_EmptyPassword(t *testing.T) {
	svc := newTestAuthService(&mocks.AuthRepo{})
	_, status, err := svc.Register("alice_user", "alice@example.com", "")
	assert.Equal(t, 400, status)
	assert.ErrorContains(t, err, "required")
}

func TestAuthService_Register_ShortPassword(t *testing.T) {
	svc := newTestAuthService(&mocks.AuthRepo{})
	_, status, err := svc.Register("alice_user", "alice@example.com", "abc")
	assert.Equal(t, 400, status)
	assert.ErrorContains(t, err, "6 characters")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return &models.User{ID: 99, Email: email}, nil // existing user
		},
	}
	svc := newTestAuthService(mockRepo)
	_, status, err := svc.Register("alice_user", "alice@example.com", "password123")
	assert.Equal(t, 409, status)
	assert.ErrorContains(t, err, "already exists")
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestAuthService_Login_Success(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpwd"), bcrypt.MinCost)
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return &models.User{
				ID: 2, Email: email, Role: "user", Status: "active",
				Password: sql.NullString{String: string(hashed), Valid: true},
			}, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	result, status, err := svc.Login("alice@example.com", "correctpwd")

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.NotEmpty(t, result["token"])
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpwd"), bcrypt.MinCost)
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return &models.User{
				ID: 2, Email: email, Role: "user", Status: "active",
				Password: sql.NullString{String: string(hashed), Valid: true},
			}, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	_, status, err := svc.Login("alice@example.com", "wrongpwd")
	assert.Equal(t, 401, status)
	assert.ErrorContains(t, err, "invalid credentials")
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return nil, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	_, status, err := svc.Login("nobody@example.com", "pwd")
	assert.Equal(t, 401, status)
	assert.ErrorContains(t, err, "invalid credentials")
}

func TestAuthService_Login_InactiveAccount(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("pwd"), bcrypt.MinCost)
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) {
			return &models.User{
				ID: 3, Email: email, Role: "user", Status: "suspended",
				Password: sql.NullString{String: string(hashed), Valid: true},
			}, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	_, status, err := svc.Login("banned@example.com", "pwd")
	assert.Equal(t, 403, status)
	assert.ErrorContains(t, err, "suspended")
}

func TestAuthService_Login_EmptyFields(t *testing.T) {
	svc := newTestAuthService(&mocks.AuthRepo{})
	_, status, err := svc.Login("", "")
	assert.Equal(t, 400, status)
	assert.ErrorContains(t, err, "required")
}

// ── ParseToken ────────────────────────────────────────────────────────────────

func TestAuthService_ParseToken_Valid(t *testing.T) {
	// Register to get a real token, then re-parse it.
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(email string) (*models.User, error) { return nil, nil },
		CreateLocalFn: func(name, email, _ string) (*models.User, error) {
			return &models.User{ID: 7, Username: name, Email: email, Role: "user", Status: "active",
				Password: sql.NullString{Valid: true}}, nil
		},
	}
	svc := newTestAuthService(mockRepo)
	result, _, err := svc.Register("bob_user", "bob@example.com", "password123")
	require.NoError(t, err)

	token := result["token"].(string)
	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, 7, claims.UserID)
	assert.Equal(t, "bob@example.com", claims.Email)
}

func TestAuthService_ParseToken_Invalid(t *testing.T) {
	svc := newTestAuthService(&mocks.AuthRepo{})
	_, err := svc.ParseToken("this.is.garbage")
	assert.Error(t, err)
}

func TestAuthService_ParseToken_SignedWithDifferentSecret(t *testing.T) {
	// Token issued by a different service (different secret) must be rejected.
	otherSvc := &AuthService{repo: &mocks.AuthRepo{}, jwtSecret: []byte("totally-different-secret!!!!!!")}
	mockRepo := &mocks.AuthRepo{
		FindByEmailFn: func(string) (*models.User, error) { return nil, nil },
		CreateLocalFn: func(name, email, _ string) (*models.User, error) {
			return &models.User{ID: 1, Username: name, Email: email, Role: "user", Status: "active",
				Password: sql.NullString{Valid: true}}, nil
		},
	}
	issuerSvc := newTestAuthService(mockRepo)
	result, _, _ := issuerSvc.Register("xuser1", "x@x.com", "password123")
	require.NotNil(t, result, "Register must succeed to produce a token")
	token := result["token"].(string)

	_, err := otherSvc.ParseToken(token)
	assert.Error(t, err)
}

// ── GoogleAuthURL ─────────────────────────────────────────────────────────────

func TestAuthService_GoogleAuthURL_ContainsGoogleDomain(t *testing.T) {
	svc := &AuthService{
		repo:      &mocks.AuthRepo{},
		jwtSecret: testJWTSecret,
		google: GoogleConfig{
			ClientID:    "client-id",
			RedirectURI: "http://localhost/callback",
		},
	}
	url := svc.GoogleAuthURL("web")
	assert.Contains(t, url, "accounts.google.com")
	assert.Contains(t, url, "state=web")
}
