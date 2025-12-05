package cmd

import (
	"fmt"
	"log"

	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

// containersStartCmd represents the start command
var containersStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all stopped Ghostwriter services",
	Long: `Start all stopped Ghostwriter services. This performs the equivalent
of running the "docker compose start" command.

Production containers are targeted by default. Use the "--dev" flag to
target development containers`,
	Run: containersStart,
}

func init() {
	containersCmd.AddCommand(containersStartCmd)
}

func containersStart(cmd *cobra.Command, args []string) {
	dockerInterface := docker.GetDockerInterface(mode)
	if dockerInterface.UseDevInfra {
		fmt.Println("[+] Starting the development environment")
	} else {
		fmt.Println("[+] Starting the production environment")
	}

	startErr := dockerInterface.RunComposeCmd("start")
	if startErr != nil {
		log.Fatalf("Error trying to restart the containers with %s: %v\n", dockerInterface.ComposeFile, startErr)
	}
}
