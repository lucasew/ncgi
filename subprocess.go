package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const PortPlaceholder = "%PORT%"

// handleSubprocess handles running the passed command args.
func handleSubprocess(ctx context.Context, port int, args ...string) error {
	var err error
	if len(args) == 0 {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}
	}
	args[0], err = exec.LookPath(args[0])
	if err != nil {
		return err
	}
	for i := range args {
		args[i] = strings.ReplaceAll(args[i], PortPlaceholder, fmt.Sprintf("%d", port))
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	time.Sleep(1 * time.Second)
	return cmd.Run()
}
