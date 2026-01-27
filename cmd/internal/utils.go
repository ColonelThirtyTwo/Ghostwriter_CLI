package internal

// Various utilities used by other parts of the internal package
// Includes utilities for interacting with the file system

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetCwdFromExe gets the current working directory based on "ghostwriter-cli" location.
func GetCwdFromExe() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get path to current executable")
	}
	return filepath.Dir(exe)
}

// FileExists determines if a given string is a valid filepath.
// Reference: https://golangcode.com/check-if-a-file-exists/
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return !info.IsDir()
}

// DirExists determines if a given string is a valid directory.
// Reference: https://golangcode.com/check-if-a-file-exists/
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return info.IsDir()
}

// CheckPath checks the $PATH environment variable for a given "cmd" and return a "bool"
// indicating if it exists.
func CheckPath(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetLocalGhostwriterVersion fetches the local Ghostwriter version from the "VERSION" file.
func GetLocalGhostwriterVersion() (string, error) {
	var output string

	versionFile := filepath.Join(GetCwdFromExe(), "VERSION")
	if FileExists(versionFile) {
		file, err := os.Open(versionFile)
		if err != nil {
			return output, err
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return output, err
		}

		output = fmt.Sprintf("Ghostwriter %s (%s)", lines[0], lines[1])
	} else {
		output = "Could not read Ghostwriter's `VERSION` file"
	}

	return output, nil
}

// GetRemoteVersion fetches the latest version information from GitHub's API for the given repository.
func GetRemoteVersion(owner string, repository string) (string, string, error) {
	baseUrl := "https://api.github.com/repos/" + owner + "/" + repository + "/releases/latest"
	client := http.Client{Timeout: time.Second * 10}
	resp, err := client.Get(baseUrl)
	if err != nil {
		return "", "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", "", readErr
	}

	var githubJson map[string]interface{}
	jsonErr := json.Unmarshal(body, &githubJson)
	if jsonErr != nil {
		return "", "", jsonErr
	}

	tagName := githubJson["tag_name"].(string)
	url := githubJson["html_url"].(string)
	return tagName, url, nil
}

// Contains checks if a slice of strings ("slice" parameter) contains a given
// string ("search" parameter).
func Contains(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

// Silence any output from tests.
// Place `defer quietTests()()` after test declarations.
// Ref: https://stackoverflow.com/a/58720235
func quietTests() func() {
	null, _ := os.Open(os.DevNull)
	sout := os.Stdout
	serr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	log.SetOutput(null)
	return func() {
		defer null.Close()
		os.Stdout = sout
		os.Stderr = serr
		log.SetOutput(os.Stderr)
	}
}

// AskForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
// Original source: https://gist.github.com/r0l1/3dcbb0c8f6cfe9c66ab8008f55f8f28b
func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
