package tool

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

func NewCommandTool() *gollm.FunctionDefinition {
	return &gollm.FunctionDefinition{
		Name:        "exec-command",
		Description: "Execute shell commands and return their output.",
		Parameters: &gollm.Schema{
			Type: gollm.TypeObject,
			Properties: map[string]*gollm.Schema{
				"command": {
					Type:        gollm.TypeString,
					Description: "The full shell command to execute, e.g., 'gcloud compute instances list'.",
				},
			},
			Required: []string{"command"},
		},
	}
}

func ExecuteCommand(ctx context.Context, command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	gcloudPath := filepath.Join(os.Getenv("HOME"), "google-cloud-sdk", "bin")
	currentPath := os.Getenv("PATH")
	if currentPath != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", gcloudPath, currentPath))
	} else {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s", gcloudPath))
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()

	lines := strings.Split(out.String(), "\n")

	maxLines := 20
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	limitedOutput := "Showing up to the 20 most recent lines:\n" + strings.Join(lines, "\n")

	return limitedOutput, err
}
