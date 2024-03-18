package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
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
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	config = &oauth2.Config{
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
	}
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search for anime on MyAnimeList",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			query := args[0]
			limit := cmd.Flag("limit").Value.String()

			tokenFile, err := getTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			token, err := readTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := searchAnime(query, limit, token.AccessToken)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Search Results:")
			for _, anime := range result.Data {
				fmt.Printf("%s\n", anime.Node.Title)
			}
		},
	}
}

func userListCmd() *cobra.Command {
	return &cobra.Command{
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

			token, err := readTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := getUserAnimeList(username, limit, token.AccessToken)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("User Anime List:")
			for _, anime := range result.Data {
				fmt.Printf("%s\n", anime.Node.Title)
			}
		},
	}
}

func getTokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, anitrackTokenFileName), nil
}

func getToken() (*oauth2.Token, error) {
	codeVerifier, codeChallenge := generateCodeVerifierAndChallenge()

	url := config.AuthCodeURL("state", oauth2.SetAuthURLParam("code_challenge", codeChallenge))

	fmt.Printf("Please visit the following URL to login: \n%s\n", url)
	fmt.Println("After successful login, please enter the code here: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}

	return exchangeAuthorizationCodeForToken(config, code, codeVerifier)
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

func readTokenFromFile(filePath string) (*oauth2.Token, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token oauth2.Token
	if err := json.NewDecoder(file).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func writeTokenToFile(token *oauth2.Token, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(token); err != nil {
		return err
	}

	return nil
}

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
	searchURL := fmt.Sprintf("%s/anime?q=%s&limit=%s", apiBaseURL, query, limit)
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
	userListURL := fmt.Sprintf("%s/users/%s/animelist?fields=list_status&limit=%s", apiBaseURL, username, limit)
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
