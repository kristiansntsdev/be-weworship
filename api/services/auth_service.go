package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/types"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// GoogleConfig holds Google OAuth 2.0 credentials and redirect config.
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string // Go API's own callback URL registered in Google Console
	ClientURL    string // Next.js web app URL (for post-auth redirect)
	MobileScheme string // Expo deep-link scheme, e.g. "weworship"
}

type AuthService struct {
	repo      *repositories.AuthRepository
	jwtSecret []byte
	google    GoogleConfig
}

func NewAuthService(repo *repositories.AuthRepository, jwtSecret []byte, google GoogleConfig) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret, google: google}
}

// GoogleAuthURL builds the Google consent-screen URL.
// client is "web" or "mobile" — passed through as OAuth state so the
// callback knows where to redirect after auth.
func (s *AuthService) GoogleAuthURL(client string) string {
	params := url.Values{
		"client_id":     {s.google.ClientID},
		"redirect_uri":  {s.google.RedirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"access_type":   {"offline"},
		"prompt":        {"select_account"},
		"state":         {client},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// GoogleCallback exchanges the authorisation code, finds-or-creates the user,
// issues a JWT, and returns the URL to redirect the client to.
func (s *AuthService) GoogleCallback(code, state string) (string, error) {
	// 1. Exchange code → access token
	tokenRes, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"code":          {code},
		"client_id":     {s.google.ClientID},
		"client_secret": {s.google.ClientSecret},
		"redirect_uri":  {s.google.RedirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", fmt.Errorf("google token exchange: %w", err)
	}
	defer tokenRes.Body.Close()

	var tokenJSON struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenRes.Body).Decode(&tokenJSON); err != nil || tokenJSON.AccessToken == "" {
		return "", fmt.Errorf("google token response invalid")
	}

	// 2. Fetch user info
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenJSON.AccessToken)
	userRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google userinfo fetch: %w", err)
	}
	defer userRes.Body.Close()

	var userJSON struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(userRes.Body).Decode(&userJSON); err != nil || userJSON.Email == "" {
		return "", fmt.Errorf("google userinfo response invalid")
	}

	// 3. Find or create the user in the database
	user, err := s.repo.FindOrCreateGoogleUser(userJSON.Email, userJSON.Name)
	if err != nil {
		return "", fmt.Errorf("user upsert: %w", err)
	}

	// 4. Issue a JWT
	token, err := s.issueToken(types.Claims{
		UserID:    user.ID,
		UserType:  "peserta",
		Username:  user.Email,
		UserLevel: user.UserLevel,
	})
	if err != nil {
		return "", err
	}

	// 5. Build the final redirect URL based on which client initiated the flow
	if state == "mobile" {
		return s.google.MobileScheme + "://auth/callback?token=" + token, nil
	}
	return s.google.ClientURL + "/auth/callback?token=" + token, nil
}

// GoogleLoginErrorURL returns the login page URL with an error query param.
func (s *AuthService) GoogleLoginErrorURL(errKey string) string {
	return s.google.ClientURL + "/auth/v2/login?error=" + errKey
}

func (s *AuthService) Login(username, email, password string) (map[string]any, int, error) {
	identifier := strings.TrimSpace(username)
	if identifier == "" {
		identifier = strings.TrimSpace(email)
	}
	if identifier == "" || password == "" {
		return nil, 400, fmt.Errorf("username/email and password are required")
	}

	if p, err := s.repo.FindPengurusByUsername(identifier); err != nil {
		return nil, 500, err
	} else if p != nil {
		if !matchesPassword(p.Password, password) {
			return nil, 401, fmt.Errorf("invalid credentials")
		}
		level, _ := strconv.Atoi(strings.TrimSpace(p.LevelAdmin))
		if level <= 1 {
			return nil, 403, fmt.Errorf("insufficient admin level access")
		}
		token, err := s.issueToken(types.Claims{UserID: p.ID, UserType: "pengurus", Username: p.Username})
		if err != nil {
			return nil, 500, err
		}
		return map[string]any{"token": token, "user": mapPengurus(*p)}, 200, nil
	}

	u, err := s.repo.FindPesertaByEmail(identifier)
	if err != nil {
		return nil, 500, err
	}
	if u == nil {
		return nil, 401, fmt.Errorf("invalid credentials")
	}
	if !matchesPassword(u.Password, password) {
		return nil, 401, fmt.Errorf("invalid credentials")
	}
	level, _ := strconv.Atoi(strings.TrimSpace(u.UserLevel))
	if level <= 2 {
		return nil, 403, fmt.Errorf("insufficient user level access")
	}
	if strings.TrimSpace(u.Verifikasi) != "1" {
		return nil, 403, fmt.Errorf("account not verified")
	}
	token, err := s.issueToken(types.Claims{UserID: u.ID, UserType: "peserta", Username: u.Email, UserLevel: u.UserLevel})
	if err != nil {
		return nil, 500, err
	}
	return map[string]any{"token": token, "user": mapPeserta(*u)}, 200, nil
}

func (s *AuthService) issueToken(c types.Claims) (string, error) {
	now := time.Now()
	c.RegisteredClaims = jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour))}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(s.jwtSecret)
}

func (s *AuthService) ParseToken(token string) (*types.Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &types.Claims{}, func(t *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	cl, ok := parsed.Claims.(*types.Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return cl, nil
}

func mapPengurus(p models.Pengurus) map[string]any {
	return map[string]any{"id": p.ID, "nama": p.Nama, "username": p.Username, "userType": "pengurus", "isAdmin": true, "leveladmin": p.LevelAdmin, "nowa": p.Nowa, "kotalevelup": p.Kota}
}

func mapPeserta(u models.Peserta) map[string]any {
	return map[string]any{"id": u.ID, "nama": u.Nama, "username": u.Email, "userCode": u.UserCode, "userType": "peserta", "isAdmin": false, "userlevel": u.UserLevel, "verifikasi": u.Verifikasi, "status": u.Status, "role": u.Role}
}

func matchesPassword(stored, input string) bool {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(input)) == nil
	}
	return stored == input
}
