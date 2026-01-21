package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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
	currentVersion := dockerInterface.GetVersion()
	fmt.Printf("Current version: %s\n", currentVersion)

	req, err := http.NewRequest("GET", "https://api.github.com/repos/GhostManager/Ghostwriter/releases/latest", nil)
	if err != nil {
		log.Fatalf("Could not create request: %v\n", err)
	}
	req.Header.Add("User-Agent", "Ghostwriter-CLI")
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v\n", err)
	}
	if res.StatusCode != 200 {
		log.Fatalf("Error sending request: status code %s\n", res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	var response githubReleaseResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Error parsing response body: %v\n", err)
	}
	latestVersion := response.Tag

	fmt.Printf("Latest version: %s\n", latestVersion)

	if currentVersion != latestVersion && dockerInterface.ManageComposeFile {
		fmt.Println("Run the `install` subcommand to install the latest version")
	}
}

type githubReleaseResponse struct {
	Tag string `json:"tag_name"`
}
