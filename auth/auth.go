package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const (
	AnitrackTokenFileName = ".anitrack.conf"
)

var (
	config *oauth2.Config
	server *http.Server
)

func InitializeOAuthConfig() *oauth2.Config {
	config = &oauth2.Config{
		Scopes: []string{"read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://myanimelist.net/v1/oauth2/authorize",
			TokenURL:  "https://myanimelist.net/v1/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: "http://localhost:9999/oauth/callback",
	}
	return config
}

func GetTokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, AnitrackTokenFileName), nil
}

func GetToken() (*oauth2.Token, error) {
	codeChan := make(chan string)
	defer close(codeChan)

	server = StartServer(codeChan)
	defer ShutdownServer()

	clientId := ""
	clientSecret := ""
	fmt.Print("Enter MAL client ID: ")
	fmt.Scanln(&clientId)

	fmt.Print("Enter MAL client secret: ")
	fmt.Scanln(&clientSecret)

	// Update config with user-provided client ID and client secret
	config.ClientID = clientId
	config.ClientSecret = clientSecret

	codeVerifier, codeChallenge := GenerateCodeVerifierAndChallenge()

	url := config.AuthCodeURL("state", oauth2.SetAuthURLParam("code_challenge", codeChallenge))

	fmt.Printf("Please visit the following URL to login: \n%s\n", url)
	fmt.Println("After successful login, please enter the code here: ")

	if err := browser.OpenURL(url); err != nil {
		panic(fmt.Errorf("failed to open browser for authentication %s", err.Error()))
	}
	code := <-codeChan

	return ExchangeAuthorizationCodeForToken(config, code, codeVerifier)
}

func StartServer(codeChan chan string) *http.Server {
	server := &http.Server{Addr: ":9999"}
	http.HandleFunc("/oauth/callback", handleOAuthCallback(codeChan))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	return server
}

func ShutdownServer() {
	if server != nil {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shut down server: %v", err)
		}
	}
}

func HandleOAuthCallback(codeChan chan string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParts := r.URL.Query()

		code := queryParts.Get("code")

		codeChan <- code

		msg := "<p><strong>Authentication successful</strong>. You may now close this tab.</p>"
		fmt.Fprint(w, msg)
	}
}

func ExchangeAuthorizationCodeForToken(config *oauth2.Config, code, codeVerifier string) (*oauth2.Token, error) {
	values := url.Values{}
	values.Set("client_id", config.ClientID)
	values.Set("client_secret", config.ClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", config.RedirectURL)
	values.Set("code_verifier", codeVerifier)
	values.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(config.Endpoint.TokenURL, values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var token oauth2.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func ReadAccessTokenFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var tokenData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&tokenData); err != nil {
		return "", err
	}
	accessToken, ok := tokenData["access_token"].(string)
	if !ok {
		return "", errors.New("access token not found or not a string")
	}

	return accessToken, nil
}

func WriteTokenToFile(token *oauth2.Token, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tokenData := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expiry":        token.Expiry,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokenData); err != nil {
		return err
	}

	return nil
}

func GenerateCodeVerifierAndChallenge() (string, string) {
	randomBytes := make([]byte, 64)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Fatal("Failed to generate random bytes:", err)
	}

	codeVerifier := base64.URLEncoding.EncodeToString(randomBytes)
	codeVerifier = codeVerifier[:len(codeVerifier)-2]

	return codeVerifier, codeVerifier
}
