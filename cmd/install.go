package cmd

import (
	"fmt"
	"log"

	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Builds containers and performs first-time setup of Ghostwriter",
	Long: `Builds containers and performs first-time setup of Ghostwriter. A production
environment is installed by default. Use the "--dev" flag to install a development environment.

The command performs the following steps:

* Sets up the default server configuration
* Generates TLS certificates for the server
* Builds the Docker containers
* Creates a default admin user with a randomly generated password

This command only needs to be run once. If you run it again, you will see some errors because
certain actions (e.g., creating the default user) can and should only be done once.`,
	Run: installGhostwriter,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func installGhostwriter(cmd *cobra.Command, args []string) {
	dockerInterface := docker.GetDockerInterface(mode)
	dockerInterface.Env.Save()
	if dockerInterface.UseDevInfra {
		fmt.Println("[+] Starting development environment installation")
	} else {
		fmt.Println("[+] Starting production environment installation")
		docker.GenerateCertificatePackage(dockerInterface.Dir)
	}

	buildErr := dockerInterface.RunComposeCmd("build")
	if buildErr != nil {
		log.Fatalf("Error trying to build with %s: %v\n", dockerInterface.ComposeFile, buildErr)
	}

	fmt.Println("[+] Migrating database...")
	err := dockerInterface.RunDjangoManageCommand("migrate")
	if err != nil {
		log.Fatalf("Error migrating database: %s\n", err)
	}

	fmt.Println("[+] Proceeding with Django database setup...")
	seedErr := dockerInterface.RunComposeCmd("run", "--rm", "django", "/seed_data")
	if seedErr != nil {
		log.Fatalf("Error trying to seed the database: %v\n", seedErr)
	}
	fmt.Println("[+] Proceeding with Django superuser creation...")
	userErr := dockerInterface.RunComposeCmd("run", "--rm", "django", "python", "manage.py", "createsuperuser", "--noinput", "--role", "admin")
	// This may fail if the user has already created a superuser, so we don't exit
	if userErr != nil {
		log.Printf("Error trying to create a superuser: %v\n", userErr)
		log.Println("Error may occur if you've run `install` before or made a superuser manually")
	}

	fmt.Println("[+] Starting containers...")
	err = dockerInterface.Up()
	if err != nil {
		log.Fatalf("Error bringing containers up: %s\n", err)
	}

	fmt.Println("[+] Ghostwriter is ready to go!")
	fmt.Printf("[+] You can login as `%s` with this password: %s\n", dockerInterface.Env.Get("django_superuser_username"), dockerInterface.Env.Get("django_superuser_password"))
	fmt.Println("[+] You can get your admin password by running: ghostwriter-cli config get admin_password")
}
