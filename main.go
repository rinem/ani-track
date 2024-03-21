package main

import (
	"log"

	"github.com/rinem/ani-track/cmd"
	"github.com/spf13/cobra"
)

const (
	anitrackTokenFileName = ".anitrack.conf"
	apiBaseURL            = "https://api.myanimelist.net/v2/"
)

func main() {
	rootCmd := &cobra.Command{Use: "ani-track"}
	rootCmd.AddCommand(cmd.LoginCmd(), cmd.SearchCmd(), cmd.UserListCmd())
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
