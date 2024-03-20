package main

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
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	anitrackTokenFileName = ".anitrack.conf"
	apiBaseURL            = "https://api.myanimelist.net/v2/"
)

var (
	config *oauth2.Config
	token  *oauth2.Token
	server *http.Server
)

func main() {

	config = &oauth2.Config{
		Scopes: []string{"read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://myanimelist.net/v1/oauth2/authorize",
			TokenURL:  "https://myanimelist.net/v1/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: "http://localhost:9999/oauth/callback",
	}

	rootCmd := &cobra.Command{Use: "ani-track"}
	rootCmd.AddCommand(loginCmd(), searchCmd(), userListCmd())
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Perform OAuth login to MyAnimeList",
		Run: func(cmd *cobra.Command, args []string) {
			tokenFile, err := getTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			token, err = getToken()
			if err != nil {
				log.Fatal(err)
			}

			if err := writeTokenToFile(token, tokenFile); err != nil {
				log.Fatal(err)
			}

			fmt.Println("Login successful. Token saved.")
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			shutdownServer()
		},
	}
}

func searchCmd() *cobra.Command {
	var limit string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for anime on MyAnimeList",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			query := args[0]

			tokenFile, err := getTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			accessToken, err := readAccessTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := searchAnime(query, limit, accessToken)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Search Results:")
			for _, anime := range result.Data {
				fmt.Printf("%s\n", anime.Node.Title)
			}
		},
	}

	cmd.Flags().StringVarP(&limit, "limit", "l", "5", "Limit search results")

	return cmd
}

func userListCmd() *cobra.Command {
	var limit string
	cmd := &cobra.Command{
		Use:   "userlist [username]",
		Short: "Get anime list of a user from MyAnimeList",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]

			limit := cmd.Flag("limit").Value.String()

			tokenFile, err := getTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			token, err := readAccessTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := getUserAnimeList(username, limit, token)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("User Anime List:")
			for _, anime := range result.Data {
				fmt.Printf("%s\n", anime.Node.Title)
			}
		},
	}

	cmd.Flags().StringVarP(&limit, "limit", "l", "5", "Limit results")

	return cmd
}

func getTokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, anitrackTokenFileName), nil
}

func getToken() (*oauth2.Token, error) {
	codeChan := make(chan string)
	defer close(codeChan)

	server = startServer(codeChan)
	defer shutdownServer()

	// Get client id and secret from env if present else ask user
	clientId := ""
	clientSecret := ""
	fmt.Print("Enter MAL client ID: ")
	fmt.Scanln(&clientId)

	fmt.Print("Enter MAL client secret: ")
	fmt.Scanln(&clientSecret)

	// Update config with user-provided client ID and client secret
	config.ClientID = clientId
	config.ClientSecret = clientSecret

	codeVerifier, codeChallenge := generateCodeVerifierAndChallenge()

	url := config.AuthCodeURL("state", oauth2.SetAuthURLParam("code_challenge", codeChallenge))

	fmt.Printf("Please visit the following URL to login: \n%s\n", url)
	fmt.Println("After successful login, please enter the code here: ")

	if err := browser.OpenURL(url); err != nil {
		panic(fmt.Errorf("failed to open browser for authentication %s", err.Error()))
	}
	code := <-codeChan

	return exchangeAuthorizationCodeForToken(config, code, codeVerifier)
}

func startServer(codeChan chan string) *http.Server {
	server := &http.Server{Addr: ":9999"}
	http.HandleFunc("/oauth/callback", handleOAuthCallback(codeChan))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	return server
}

func shutdownServer() {
	if server != nil {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shut down server: %v", err)
		}
	}
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

func exchangeAuthorizationCodeForToken(config *oauth2.Config, code, codeVerifier string) (*oauth2.Token, error) {
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

func readAccessTokenFromFile(filePath string) (string, error) {
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

func writeTokenToFile(token *oauth2.Token, filePath string) error {
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

// func writeTokenToFile(token *oauth2.Token, filePath string) error {
// 	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()
//
// 	_, err = fmt.Fprintf(file, "%s\n", token)
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }

func generateCodeVerifierAndChallenge() (string, string) {
	randomBytes := make([]byte, 64)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Fatal("Failed to generate random bytes:", err)
	}

	codeVerifier := base64.URLEncoding.EncodeToString(randomBytes)
	codeVerifier = codeVerifier[:len(codeVerifier)-2]

	return codeVerifier, codeVerifier
}

type AnimeSearchResult struct {
	Data []struct {
		Node struct {
			Title string `json:"title"`
		} `json:"node"`
	} `json:"data"`
}

func searchAnime(query, limit, accessToken string) (*AnimeSearchResult, error) {
	searchURL := fmt.Sprintf("%sanime?q=%s&limit=%s", apiBaseURL, query, limit)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AnimeSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type UserAnimeListResult struct {
	Data []struct {
		Node struct {
			Title string `json:"title"`
		} `json:"node"`
	} `json:"data"`
}

func getUserAnimeList(username, limit, accessToken string) (*UserAnimeListResult, error) {
	userListURL := fmt.Sprintf("%susers/%s/animelist?fields=list_status&limit=%s", apiBaseURL, username, limit)
	req, err := http.NewRequest("GET", userListURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserAnimeListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
