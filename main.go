package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Arguments: owner project")
		return errors.New("invalid arguments")
	}

	owner := os.Args[1]
	project := os.Args[2]

	oldVersion, newVersion, err := getVersions(owner, project)
	if err != nil {
		return fmt.Errorf("error checking versions in diff: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, project, oldVersion, newVersion)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get failed: %w", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read all error: %w", err)
	}

	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Bump github.com/%s/%s\n", owner, project))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("%s -> %s\n", oldVersion, newVersion))
	builder.WriteString("\n")
	builder.WriteString("Commits:\n")

	for i, commit := range response.ResponseCommits {
		if i > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("  - %s\n", commit.Sha))
		sc := bufio.NewScanner(strings.NewReader(commit.Commit.Message))
		for sc.Scan() {
			builder.WriteString("    " + sc.Text() + "\n")
			break
		}
	}

	fmt.Println(builder.String())
	return nil
}

type Response struct {
	ResponseCommits []ResponseCommit `json:"commits"`
}

type ResponseCommit struct {
	Sha    string               `json:"sha"`
	Commit ResponseCommitCommit `json:"commit"`
}

type ResponseCommitCommit struct {
	Message string `json:"message"`
}

func getVersions(owner, project string) (string, string, error) {
	var removed, added string

	packageName := fmt.Sprintf(`github.com/%s/%s`, owner, project)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if !strings.Contains(text, packageName) {
			continue
		}
		if strings.HasPrefix(text, "+	") {
			added = text
		}
		if strings.HasPrefix(text, "-	") {
			removed = text
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("scan failed: %w", err)
	}

	oldVersion, err := getVersion(removed)
	if err != nil {
		return "", "", fmt.Errorf("error getting old version: %w", err)
	}

	newVersion, err := getVersion(added)
	if err != nil {
		return "", "", fmt.Errorf("error getting new version: %w", err)
	}

	return oldVersion, newVersion, nil
}

func getVersion(line string) (string, error) {
	line = strings.TrimPrefix(line, "+")
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)

	parts := strings.Fields(line)
	if len(parts) != 2 {
		return "", errors.New("something went wrong with splitting into parts")
	}

	pseudoVersionOrVersion := strings.Split(parts[1], "-")
	if len(pseudoVersionOrVersion) == 3 {
		hash := pseudoVersionOrVersion[2]
		if len(hash) != 12 {
			return hash, errors.New("version information not found")
		}
		return hash, nil
	}

	return "", errors.New("invalid length")
}
