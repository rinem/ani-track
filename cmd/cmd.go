package cmd

import (
	"fmt"
	"log"

	"github.com/rinem/ani-track/api"
	"github.com/rinem/ani-track/auth"
	"github.com/spf13/cobra"
)

func LoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Perform OAuth login to MyAnimeList",
		Run: func(cmd *cobra.Command, args []string) {
			tokenFile, err := auth.GetTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			token, err := auth.GetToken()
			if err != nil {
				log.Fatal(err)
			}

			if err := auth.WriteTokenToFile(token, tokenFile); err != nil {
				log.Fatal(err)
			}

			fmt.Println("Login successful. Token saved.")
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			auth.ShutdownServer()
		},
	}
}

func SearchCmd() *cobra.Command {
	var limit string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for anime on MyAnimeList",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			query := args[0]

			tokenFile, err := auth.GetTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			accessToken, err := auth.ReadAccessTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := api.SearchAnime(query, limit, accessToken)
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

func UserListCmd() *cobra.Command {
	var limit string
	cmd := &cobra.Command{
		Use:   "userlist [username]",
		Short: "Get anime list of a user from MyAnimeList",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]

			limit := cmd.Flag("limit").Value.String()

			tokenFile, err := auth.GetTokenFilePath()
			if err != nil {
				log.Fatal(err)
			}

			token, err := auth.ReadAccessTokenFromFile(tokenFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := api.GetUserAnimeList(username, limit, token)
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
