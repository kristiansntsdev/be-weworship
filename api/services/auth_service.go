package services

import (
"encoding/json"
"fmt"
"net/http"
"net/url"
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
ClientURL    string // Web app URL (for post-auth redirect)
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
// client is "web" or "mobile".
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

req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
req.Header.Set("Authorization", "Bearer "+tokenJSON.AccessToken)
userRes, err := http.DefaultClient.Do(req)
if err != nil {
return "", fmt.Errorf("google userinfo fetch: %w", err)
}
defer userRes.Body.Close()

var userJSON struct {
ID    string `json:"id"`
Email string `json:"email"`
Name  string `json:"name"`
}
if err := json.NewDecoder(userRes.Body).Decode(&userJSON); err != nil || userJSON.Email == "" {
return "", fmt.Errorf("google userinfo response invalid")
}

user, err := s.repo.FindOrCreateGoogleUser(userJSON.Email, userJSON.Name, userJSON.ID)
if err != nil {
return "", fmt.Errorf("user upsert: %w", err)
}

token, err := s.issueToken(user)
if err != nil {
return "", err
}

if state == "mobile" {
scheme := s.google.MobileScheme
if scheme == "" {
scheme = "weworship"
}
return scheme + "://auth/callback?token=" + token, nil
}
if s.google.ClientURL != "" {
return s.google.ClientURL + "/auth/callback?token=" + token, nil
}
return "/?token=" + token, nil
}

// GoogleLoginErrorURL returns the login page URL with an error query param.
func (s *AuthService) GoogleLoginErrorURL(errKey string) string {
if s.google.ClientURL != "" {
return s.google.ClientURL + "/auth/v2/login?error=" + errKey
}
return "/?error=" + errKey
}

// Register creates a new local email+password account and returns a JWT.
func (s *AuthService) Register(name, email, password string) (map[string]any, int, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" || password == "" {
		return nil, 400, fmt.Errorf("name, email and password are required")
	}
	if len(password) < 6 {
		return nil, 400, fmt.Errorf("password must be at least 6 characters")
	}
	existing, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, 500, err
	}
	if existing != nil {
		return nil, 409, fmt.Errorf("an account with that email already exists")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, 500, fmt.Errorf("failed to hash password")
	}
	u, err := s.repo.CreateLocal(name, email, string(hashed))
	if err != nil {
		return nil, 500, err
	}
	token, err := s.issueToken(u)
	if err != nil {
		return nil, 500, err
	}
	return map[string]any{"token": token, "user": mapUser(*u)}, 201, nil
}

// Login authenticates a user by email + password.
func (s *AuthService) Login(email, password string) (map[string]any, int, error) {
email = strings.TrimSpace(email)
if email == "" || password == "" {
return nil, 400, fmt.Errorf("email and password are required")
}

u, err := s.repo.FindByEmail(email)
if err != nil {
return nil, 500, err
}
if u == nil || !u.Password.Valid {
return nil, 401, fmt.Errorf("invalid credentials")
}
if !matchesPassword(u.Password.String, password) {
return nil, 401, fmt.Errorf("invalid credentials")
}
if u.Status != "active" {
return nil, 403, fmt.Errorf("account is %s", u.Status)
}

token, err := s.issueToken(u)
if err != nil {
return nil, 500, err
}
return map[string]any{"token": token, "user": mapUser(*u)}, 200, nil
}

func (s *AuthService) issueToken(u *models.User) (string, error) {
now := time.Now()
c := types.Claims{
UserID: u.ID,
Role:   u.Role,
Name:   u.Name,
Email:  u.Email,
RegisteredClaims: jwt.RegisteredClaims{
IssuedAt:  jwt.NewNumericDate(now),
ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
},
}
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

func mapUser(u models.User) map[string]any {
var avatarURL any
if u.AvatarURL.Valid {
avatarURL = u.AvatarURL.String
}
return map[string]any{
"id":         u.ID,
"name":       u.Name,
"email":      u.Email,
"role":       u.Role,
"provider":   u.Provider,
"verified":   u.Verified,
"status":     u.Status,
"avatar_url": avatarURL,
}
}

func matchesPassword(stored, input string) bool {
if strings.HasPrefix(stored, "$2") {
return bcrypt.CompareHashAndPassword([]byte(stored), []byte(input)) == nil
}
return stored == input
}
