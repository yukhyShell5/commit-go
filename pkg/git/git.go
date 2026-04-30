package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func GetGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	diff := out.String()
	if diff == "" {
		return "", fmt.Errorf("no staged changes found. Use 'git add' to stage files")
	}
	return diff, nil
}

func ExecuteCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
