package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Arguments: owner project")
		return
	}

	owner := os.Args[1]
	project := os.Args[2]
	oldVersion := getVersion(owner, project, "-")
	newVersion := getVersion(owner, project, "+")

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, project, oldVersion, newVersion)

	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Bump github.com/%s/%s (%s -> %s)\n", owner, project, oldVersion, newVersion))
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

func getVersion(owner, project, ch string) string {
	cmd := exec.Command(
		"/bin/bash",
		"-c",
		fmt.Sprintf(`git diff --cached go.mod | grep "github.com/%s/%s" | grep -- "%s	" | cut -f"2" | cut -d" " -f"2" | cut -d"-" -f"3"`, owner, project, ch),
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(out.String())
}
