package cmd

import (
	"fmt"
	"log"

	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Runs Ghostwriter's unit tests in the development environment",
	Long: `Runs Ghostwriter's unit tests in the development environment.

Requires to "install --dev" to have been run first.`,
	Run: runUnitTests,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runUnitTests(cmd *cobra.Command, args []string) {
	dockerInterface := docker.GetDockerInterface(dev)
	fmt.Println("[+] Running Ghostwriter's unit and integration tests...")

	// Save the current env values we're about to change
	currentActionSecret := docker.GhostEnv.Get("HASURA_GRAPHQL_ACTION_SECRET")
	currentSettingsModule := docker.GhostEnv.Get("DJANGO_SETTINGS_MODULE")

	// Change env values for the test conditions
	docker.GhostEnv.Set("HASURA_GRAPHQL_ACTION_SECRET", "changeme")
	docker.GhostEnv.Set("DJANGO_SETTINGS_MODULE", "config.settings.local")
	docker.WriteGhostwriterEnvironmentVariables()

	// Run the unit tests
	testErr := dockerInterface.RunDjangoManageCommand("test")
	if testErr != nil {
		log.Fatalf("Error trying to run Ghostwriter's tests: %v\n", testErr)
	}

	// Reset the changed env values
	docker.GhostEnv.Set("HASURA_GRAPHQL_ACTION_SECRET", currentActionSecret)
	docker.GhostEnv.Set("DJANGO_SETTINGS_MODULE", currentSettingsModule)
	docker.WriteGhostwriterEnvironmentVariables()
}
