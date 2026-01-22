package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	docker "github.com/GhostManager/Ghostwriter_CLI/cmd/internal"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs/updates and sets up Ghostwriter",
	Long: `Installs/updates and sets up Ghostwriter. By default, Ghostwriter will download and
install the latest version to an application data directory - use the "--mode" option to use a
source checkout instead.

The command performs the following steps:

* Sets up the default server configuration
* Generates TLS certificates for the server
* Fetches or builds the Docker containers
* Creates a default admin user with a randomly generated password

Running after initial installation will keep the existing configuration but fetch a new version
(for --mode=production) or rebuild the containers (for --mode=local-*)
`,
	Run: installGhostwriter,
}

var installVersion string

func init() {
	installCmd.PersistentFlags().StringVar(
		&installVersion,
		"version",
		"",
		"Version to install. Defaults to latest. Ignored for --mode=local-*. NOTE: downgrading is not supported.",
	)
	rootCmd.AddCommand(installCmd)
}

func installGhostwriter(cmd *cobra.Command, args []string) {
	if mode == docker.ModeProd {
		// Fetch (new) docker compose file before initializing the interface
		dir := docker.GetDockerDirFromMode(mode)
		file := "docker-compose.yml"

		fmt.Println("[+] Downloading docker-compose.yml")

		var url string
		if installVersion == "" {
			url = "https://github.com/ColonelThirtyTwo/Ghostwriter/releases/latest/download/gw-cli.yml"
		} else {
			url = "https://github.com/ColonelThirtyTwo/Ghostwriter/releases/download/" + installVersion + "/gw-cli.yml"
		}

		res, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error trying to download gw-cli.yml from GitHub: %v", err)
		}
		if res.StatusCode != 200 {
			log.Fatalf("Error trying to download gw-cli.yml from GitHub: HTTP status code %d", res.StatusCode)
		}

		buf, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Error trying to download gw-cli.yml from GitHub: %v", err)
		}

		err = os.WriteFile(
			filepath.Join(dir, file),
			buf,
			0644,
		)

		if err != nil {
			log.Fatalf("Error trying to download gw-cli.yml from GitHub: %v", err)
		}
	}

	// Get interface
	dockerInterface := docker.GetDockerInterface(mode)
	dockerInterface.Env.Save()
	if dockerInterface.UseDevInfra {
		fmt.Println("[+] Starting development environment installation")
	} else {
		fmt.Println("[+] Starting production environment installation")
		docker.GenerateCertificatePackage(dockerInterface.Dir)
	}

	// Build/pull
	var err error
	if dockerInterface.ManageComposeFile {
		fmt.Println("[+] Pulling containers...")
		err = dockerInterface.RunComposeCmd("pull")
		if err != nil {
			log.Fatalf("Error trying to pull with %s: %v\n", dockerInterface.ComposeFile, err)
		}
	} else {
		fmt.Println("[+] Building containers...")
		err = dockerInterface.RunComposeCmd("build", "--pull")
		if err != nil {
			log.Fatalf("Error trying to BUILD with %s: %v\n", dockerInterface.ComposeFile, err)
		}
	}

	fmt.Println("[+] Migrating database...")
	err = dockerInterface.RunDjangoManageCommand("migrate")
	if err != nil {
		log.Fatalf("Error migrating database: %s\n", err)
	}

	fmt.Println("[+] Proceeding with Django database setup...")
	seedErr := dockerInterface.RunComposeCmd("run", "--rm", "django", "/seed_data")
	if seedErr != nil {
		log.Fatalf("Error trying to seed the database: %v\n", seedErr)
	}
	fmt.Println("[+] Proceeding with Django superuser creation...")
	userErr := dockerInterface.RunDjangoManageCommand("createsuperuser", "--noinput", "--role", "admin")
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
