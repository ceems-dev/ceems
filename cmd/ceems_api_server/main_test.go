package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var binary, _ = filepath.Abs("../../bin/ceems_api_server")

const (
	address = "localhost:19020"
)

func TestBatchjobStatsExecutable(t *testing.T) {
	_, err := os.Stat(binary)
	if err != nil {
		t.Skipf("ceems_api_server binary not available, try to run `make build` first: %s", err)
	}

	usagestats := exec.CommandContext(
		t.Context(),
		binary,
		"--web.listen-address", address,
		"--no-security.drop-privileges",
	)
	require.NoError(t, runCommandAndTests(usagestats))
}

func runCommandAndTests(cmd *exec.Cmd) error {
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Sleep for a while and kill process
	time.Sleep(1 * time.Second)

	cmd.Process.Kill()

	return nil
}
