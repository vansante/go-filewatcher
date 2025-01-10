package watcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func RunCommand(ctx context.Context, command string, wait bool) (*exec.Cmd, error) {
	_, _ = fmt.Fprintf(os.Stderr, "--- Running: %s\n", command)

	// Run the command using `sh -c <command>` to allow for
	// shell syntax such as pipes and boolean operators
	cmd := exec.CommandContext(ctx, "sh", []string{"-c", command}...)
	// Make sure we kill child processes: https://stackoverflow.com/a/29552044
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !wait {
		err := cmd.Start()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "--- Error: %v\n", err)
			return nil, err
		}
		return cmd, nil
	}

	err := cmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "--- Error: %v\n", err)
		return nil, err
	}
	return cmd, nil
}
