package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const tokenFileName = ".anitrack.conf"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	config := &oauth2.Config{
		ClientID:     os.Getenv("MAL_CLIENT_ID"),
		ClientSecret: os.Getenv("MAL_CLIENT_SECRET"),
		Scopes:       []string{"read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://myanimelist.net/v1/oauth2/authorize",
			TokenURL:  "https://myanimelist.net/v1/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: os.Getenv("REDIRECT_URL"),
	}

	server := &http.Server{Addr: ":9999"}

	codeChan := make(chan string)

	http.HandleFunc("/oauth/callback", handleOAuthCallback(codeChan))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	codeVerifier, codeChallenge := generateCodeVerifierAndChallenge()

	url := config.AuthCodeURL("state", oauth2.SetAuthURLParam("code_challenge", codeChallenge))

	fmt.Printf("Your browser has been opened to visit::\n%s\n", url)

	if err := browser.OpenURL(url); err != nil {
		panic(fmt.Errorf("failed to open browser for authentication %s", err.Error()))
	}

	code := <-codeChan

	token, err := exchangeAuthorizationCodeForToken(config, code, codeVerifier)
	if err != nil {
		log.Fatalf("Failed to exchange authorization code for token: %v", err)
	}

	if !token.Valid() {
		log.Fatalf("Cannot get source information without accessToken: %v", err)
		return
	}

	if err := writeTokenToFile(token); err != nil {
		log.Fatalf("Failed to write token to file: %v", err)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shut down server: %v", err)
	}

	log.Println(color.CyanString("Authentication successful"))
}

func handleOAuthCallback(codeChan chan string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParts := r.URL.Query()

		code := queryParts.Get("code")

		codeChan <- code

		msg := "<p><strong>Authentication successful</strong>. You may now close this tab.</p>"
		fmt.Fprint(w, msg)
	}
}

func writeTokenToFile(token *oauth2.Token) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine user's home directory: %v", err)
	}

	tokenFilePath := filepath.Join(homeDir, tokenFileName)

	file, err := os.OpenFile(tokenFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to create token file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(token); err != nil {
		return fmt.Errorf("unable to write token to file: %v", err)
	}

	return nil
}

func generateCodeVerifierAndChallenge() (string, string) {
	randomBytes := make([]byte, 64)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Failed to generate random bytes:", err)
	}

	codeVerifier := base64.URLEncoding.EncodeToString(randomBytes)
	codeVerifier = codeVerifier[:len(codeVerifier)-2]

	return codeVerifier, codeVerifier
}

func exchangeAuthorizationCodeForToken(config *oauth2.Config, code, codeVerifier string) (*oauth2.Token, error) {
	values := url.Values{}
	values.Set("client_id", config.ClientID)
	values.Set("client_secret", config.ClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", os.Getenv("REDIRECT_URL"))
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
