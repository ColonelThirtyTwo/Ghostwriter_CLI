package cmd

import (
	"fmt"
	"log"

	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

// containersUpCmd represents the up command
var containersUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Build, (re)create, and start all Ghostwriter containers",
	Long: `Build, (re)create, and start all Ghostwriter containers. This
performs the equivalent of running the "docker compose up" command.

Production containers are targeted by default. Use the "--dev" flag to
target development containers`,
	Run: containersUp,
}

func init() {
	containersCmd.AddCommand(containersUpCmd)
}

func containersUp(cmd *cobra.Command, args []string) {
	dockerInterface := docker.GetDockerInterface(mode)
	if dockerInterface.UseDevInfra {
		fmt.Println("[+] Bringing up the development environment")
	} else {
		fmt.Println("[+] Bringing up the production environment")
	}
	dockerInterface.Env.Save()
	err := dockerInterface.Up()
	if err != nil {
		log.Fatalf("Error trying to bring up the containers with %s: %v\n", dockerInterface.ComposeFile, err)
	}
}
