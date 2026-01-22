package cmd

import (
	"fmt"
	"log"

	"github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

var checkVersionCmd = &cobra.Command{
	Use:   "check-version",
	Short: "Checks for updates",
	Long: `Checks for updates.

Prints the currently installed version of Ghostwriter and the latest released version.
The production environment is targeted by default. Use the "--mode" argument to query a development environment.`,
	Run: checkVersion,
}

func init() {
	rootCmd.AddCommand(checkVersionCmd)
}

func checkVersion(cmd *cobra.Command, args []string) {
	dockerInterface := docker.GetDockerInterface(mode)
	currentVersion, err := dockerInterface.GetVersion()
	if err != nil {
		log.Fatalf("Could not get current version: %v\n", err)
	}
	fmt.Printf("Current version: %s\n", currentVersion)

	latestVersion, err := internal.FetchLatestRelease()
	if err != nil {
		log.Fatalf("Could not get latest version: %v\n", err)
	}
	fmt.Printf("Latest version: %s\n", latestVersion)

	if currentVersion != latestVersion && dockerInterface.ManageComposeFile {
		fmt.Println("Run the `install` subcommand to install the latest version")
	}
}
