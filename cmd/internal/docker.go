package internal

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Vars for tracking the list of Ghostwriter images
// Used for filtering the list of containers returned by the Docker client
var (
	ProdImages = []string{
		"ghostwriter_production_django", "ghostwriter_production_nginx",
		"ghostwriter_production_redis", "ghostwriter_production_postgres",
		"ghostwriter_production_graphql", "ghostwriter_production_queue",
		"ghostwriter_production_collab_server",
	}
	DevImages = []string{
		"ghostwriter_local_django", "ghostwriter_local_redis",
		"ghostwriter_local_postgres", "ghostwriter_local_graphql",
		"ghostwriter_local_queue", "ghostwriter_local_collab_server",
		"ghostwriter_local_frontend",
	}
)

type DockerInterface struct {
	// Directory that docker compose file resides in
	Dir string
	// Docker compose file to use
	ComposeFile string
	// Command to use, either docker or podman
	command string
	// Daemon client, lazily initialized
	client *client.Client
}

func GetDockerInterface(dev bool) *DockerInterface {
	fmt.Println("[+] Checking the status of Docker and the Compose plugin...")
	// Check for ``docker`` first because it's required for everything to come
	dockerExists := CheckPath("docker")
	dockerCmd := "docker"
	if !dockerExists {
		podmanExists := CheckPath("podman")
		if podmanExists {
			fmt.Println("[+] Docker is not installed, but Podman is installed. Using Podman as a Docker alternative.")
			dockerCmd = "podman"
		} else {
			log.Fatalln("Neither Docker nor Podman is installed on this system, so please install Docker or Podman (in Docker compatibility mode) and try again.")
		}
	}

	// Check if the Docker Engine is running
	_, engineErr := exec.Command(dockerCmd, "info").Output()
	if engineErr != nil {
		if strings.Contains(strings.ToLower(engineErr.Error()), "permission denied") {
			log.Fatalf("%s is installed, but you don't have permission to talk to the daemon (Try running with sudo or adjusting your group membership)", dockerCmd)
		} else {
			log.Fatalf("%s is installed on this system, but the daemon may not be running", dockerCmd)
		}
	}

	// Check for the ``compose`` plugin as our first choice
	_, composeErr := exec.Command(dockerCmd, "compose", "version").Output()
	if composeErr != nil {
		// Check if the deprecated v1 script is installed
		composeScriptExists := CheckPath("docker-compose")
		if composeScriptExists {
			fmt.Println("[!] The deprecated `docker-compose` v1 script was detected on your system")
			fmt.Println("[!] Docker has deprecated v1 and this CLI tool no longer supports it")
			log.Fatalln("Please upgrade to Docker Compose v2 and try again: https://docs.docker.com/compose/install/")
		} else {
			log.Fatalln("Docker Compose is not installed, so please install it and try again: https://docs.docker.com/compose/install/")
		}
	}

	dir := GetCwdFromExe()

	file := ""
	if dev {
		file = "local.yml"
	} else {
		file = "production.yml"
	}

	// Bail out if we're not in the same directory as the YAML files
	// Otherwise, we'll get a confusing error message from the `compose` plugin
	if !FileExists(filepath.Join(dir, file)) {
		log.Fatalln("Ghostwriter CLI must be run in the same directory as the `local.yml` and `production.yml` files")
	}

	return &DockerInterface{
		Dir:         dir,
		ComposeFile: file,
		command:     dockerCmd,
	}
}

// Runs docker/podman with the specified additional arguments, in the proper CWD with the env and compose files.
// Basis for most of the other Run commands.
func (this DockerInterface) RunCmd(args ...string) error {
	path, err := exec.LookPath(this.command)
	if err != nil {
		log.Fatalf("`%s` is not installed or not available in the current PATH variable", this.command)
	}
	command := exec.Command(path, args...)
	command.Dir = this.Dir
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err = command.Start()
	if err != nil {
		log.Fatalf("Error trying to start `%s`: %v\n", this.command, err)
	}
	err = command.Wait()
	if err != nil {
		fmt.Printf("[-] Error from `%s`: %v\n", this.command, err)
		return err
	}
	return nil
}

// Similar to `RunCmd` but returns stdout
func (this DockerInterface) RunCmdWithOutput(args ...string) (string, error) {
	path, err := exec.LookPath(this.command)
	if err != nil {
		log.Fatalf("`%s` is not installed or not available in the current PATH variable", this.command)
	}
	command := exec.Command(path, args...)
	command.Dir = this.Dir
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	out, err := command.Output()
	output := string(out[:])
	return output, err
}

// Runs a `docker compose` subcommand, pointing to the configured compose file, with additional arguments.
func (this DockerInterface) RunComposeCmd(args ...string) error {
	args = append([]string{"compose", "-f", this.ComposeFile}, args...)
	return this.RunCmd(args...)
}

// / Bring all containers up
func (this DockerInterface) Up() error {
	fmt.Printf("[+] Running `%s` to bring up the containers with %s...\n", this.command, this.ComposeFile)
	return this.RunComposeCmd("up", "-d")
}

// / Take down all containers
func (this DockerInterface) Down(volumes bool) error {
	fmt.Printf("[+] Running `%s` to take down the containers with %s...\n", this.command, this.ComposeFile)
	args := []string{"down"}
	if volumes {
		args = append(args, "--volumes")
	}
	return this.RunComposeCmd(args...)
}

// Container is a custom type for storing container information similar to output from "docker containers ls".
type Container struct {
	ID     string
	Image  string
	Status string
	Ports  []container.PortSummary
	Name   string
}

// Containers is a collection of Container structs
type Containers []Container

// Len returns the length of a Containers struct
func (c Containers) Len() int {
	return len(c)
}

// Less determines if one Container is less than another Container
func (c Containers) Less(i, j int) bool {
	return c[i].Image < c[j].Image
}

// Swap exchanges the position of two Container values in a Containers struct
func (c Containers) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Gets a list of all running containers
func (this DockerInterface) GetRunning() Containers {
	var running Containers

	cli, err := this.GetDaemonClient()
	if err != nil {
		log.Fatalf("Failed to get client connection to Docker: %v", err)
	}
	containers, err := cli.ContainerList(context.Background(), client.ContainerListOptions{
		All: false,
	})
	if err != nil {
		log.Fatalf("Failed to get container list from Docker: %v", err)
	}

	for _, container := range containers.Items {
		if Contains(DevImages, container.Image) || Contains(ProdImages, container.Image) {
			running = append(running, Container{
				container.ID, container.Image, container.Status, container.Ports, container.Labels["name"],
			})
		}
	}

	return running
}

// Gets logs from a container
func (this DockerInterface) FetchLogs(containerName string, lines string) []string {
	var logs []string
	cli, err := this.GetDaemonClient()
	if err != nil {
		log.Fatalf("Failed to get client in logs: %v", err)
	}
	containers, err := cli.ContainerList(context.Background(), client.ContainerListOptions{})
	if err != nil {
		log.Fatalf("Failed to get container list: %v", err)
	}
	if len(containers.Items) > 0 {
		for _, container := range containers.Items {
			if container.Labels["name"] == containerName || containerName == "all" || container.Labels["name"] == "ghostwriter_"+containerName {
				logs = append(logs, fmt.Sprintf("\n*** Logs for `%s` ***\n\n", container.Labels["name"]))
				reader, err := cli.ContainerLogs(context.Background(), container.ID, client.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Tail:       lines,
				})
				if err != nil {
					log.Fatalf("Failed to get container logs: %v", err)
				}
				defer reader.Close()
				// Reference: https://medium.com/@dhanushgopinath/reading-docker-container-logs-with-golang-docker-engine-api-702233fac044
				p := make([]byte, 8)
				_, err = reader.Read(p)
				for err == nil {
					content := make([]byte, binary.BigEndian.Uint32(p[4:]))
					reader.Read(content)
					logs = append(logs, string(content))
					_, err = reader.Read(p)
				}
			}
		}

		if len(logs) == 0 {
			logs = append(logs, fmt.Sprintf("\n*** No logs found for requested container '%s' ***\n", containerName))
		}
	} else {
		fmt.Println("Failed to find that container")
	}
	return logs
}

// Determine if the container with the specified "name" label ("containerName" parameter) is running.
func (this DockerInterface) IsServiceRunning(containerName string) bool {
	containers := this.GetRunning()
	for _, container := range containers {
		if container.Name == strings.ToLower(containerName) {
			return true
		}
	}
	return false
}

// Determine if the Django application has completed startup based on
// the "Application startup complete" log message.
func (this DockerInterface) IsDjangoStarted() bool {
	expectedString := "Application startup complete"
	logs := this.FetchLogs("ghostwriter_django", "500")
	for _, entry := range logs {
		result := strings.Contains(entry, expectedString)
		if result {
			return true
		}
	}
	return false
}

// Check if PostgreSQL is having trouble starting due to a password mismatch.
func (this DockerInterface) IsPostgresStarted() bool {
	expectedString := "Password does not match for user"
	logs := this.FetchLogs("ghostwriter_postgres", "100")
	for _, entry := range logs {
		result := strings.Contains(entry, expectedString)
		if result {
			return true
		}
	}
	return false
}

// Determine if the Ghostwriter application has completed startup
func (this DockerInterface) WaitForDjango() bool {
	// Wait for ghostwriter to start running
	fmt.Println("[+] Waiting for Django application startup to complete...")
	counter := 0
	for {
		if !this.IsServiceRunning("ghostwriter_django") {
			fmt.Print("\n")
			log.Fatalf("Django container exited unexpectedly. Check the logs in docker for the ghostwriter_django container")
		}
		if this.IsDjangoStarted() {
			fmt.Print("\n[+] Django application started\n")
			return true
		}
		if this.IsPostgresStarted() {
			fmt.Print("\n")
			log.Fatalf("PostgreSQL cannot start because of a password mismatch. Please read: https://www.ghostwriter.wiki/getting-help/faq#ghostwriter-cli-reports-an-issue-with-postgresql")
		}

		if counter > 120 {
			fmt.Print("\n")
			log.Fatalf("Django did not start after 120 seconds.")
		}

		fmt.Print(".")
		time.Sleep(1 * time.Second)
		counter++
	}
}

// Runs the django manage.py script, with the specified arguments
func (this DockerInterface) RunDjangoManageCommand(args ...string) error {
	args = append([]string{"run", "django", "python", "manage.py"}, args...)
	return this.RunComposeCmd(args...)
}

// Connects to the docker daemon
func (this DockerInterface) GetDaemonClient() (*client.Client, error) {
	if this.client != nil {
		return this.client, nil
	}

	client, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	this.client = client
	return this.client, err
}
