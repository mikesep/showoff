package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
)

func runScript(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	signal.Ignore(os.Interrupt)

	return <-done
}
