package watcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func RunCommand(ctx context.Context, command string, wait bool) error {
	_, _ = fmt.Fprintf(os.Stderr, "--- Running: %s\n", command)

	// Run the command using `sh -c <command>` to allow for
	// shell syntax such as pipes and boolean operators
	cmd := exec.CommandContext(ctx, "sh", []string{"-c", command}...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !wait {
		err := cmd.Start()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "--- Error: %v\n", err)
			return err
		}
		return nil
	}

	err := cmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "--- Error: %v\n", err)
		return err
	}
	return nil
}
