package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

	oldVersion, err := getVersion(owner, project, "-")
	if err != nil {
		return fmt.Errorf("error checking old version in diff: %w", err)
	}

	newVersion, err := getVersion(owner, project, "+")
	if err != nil {
		return fmt.Errorf("error checking new version in diff: %w", err)
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

func getVersion(owner, project, ch string) (string, error) {
	cmd := exec.Command(
		"/bin/bash",
		"-c",
		fmt.Sprintf(`git diff --cached go.mod | grep "github.com/%s/%s" | grep -- "%s	" | cut -f"2" | cut -d" " -f"2" | cut -d"-" -f"3"`, owner, project, ch),
	)

	cmd.Stdin = os.Stdin

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	version := strings.TrimSpace(out.String())

	if len(version) != 12 {
		return "", errors.New("version information not found")
	}

	return version, nil
}
