package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/require"
)

var binary, _ = filepath.Abs("../../bin/ceems_exporter")

const (
	address = "localhost:19010"
)

func TestFileDescriptorLeak(t *testing.T) {
	_, err := os.Stat(binary)
	if err != nil {
		t.Skipf("ceems_exporter binary not available, try to run `make build` first: %s", err)
	}

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		t.Skipf(
			"proc filesystem is not available, but currently required to read number of open file descriptors: %s",
			err,
		)
	}

	_, err = fs.Stat()
	if err != nil {
		t.Errorf("unable to read process stats: %s", err)
	}

	sysfsPath, err := filepath.Abs("../../pkg/collector/testdata/sys/fs/cgroup")
	require.NoError(t, err)

	procfsPath, err := filepath.Abs("../../pkg/collector/testdata/proc")
	require.NoError(t, err)

	exporter := exec.CommandContext(
		t.Context(),
		binary,
		"--web.listen-address", address,
		"--path.cgroupfs", sysfsPath,
		"--path.procfs", procfsPath,
		"--collector.ipmi.dcmi.cmd", "pkg/collector/testdata/ipmi/freeipmi/ipmi-dcmi",
		// "--no-security.drop-privileges",
	)
	test := func(pid int) error {
		err := queryExporter(address)
		if err != nil {
			return err
		}

		proc, err := procfs.NewProc(pid)
		if err != nil {
			return err
		}

		fdsBefore, err := proc.FileDescriptors()
		if err != nil {
			return err
		}

		for range 5 {
			err := queryExporter(address)
			if err != nil {
				return err
			}
		}

		fdsAfter, err := proc.FileDescriptors()
		if err != nil {
			return err
		}

		if want, have := len(fdsBefore), len(fdsAfter); want != have {
			return fmt.Errorf(
				"want %d open file descriptors after metrics scrape, have %d",
				want,
				have,
			)
		}

		return nil
	}

	require.NoError(t, runCommandAndTests(exporter, address, test))
}

func queryExporter(address string) error {
	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", address)) //nolint:noctx
	if err != nil {
		return err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	if want, have := http.StatusOK, resp.StatusCode; want != have {
		return fmt.Errorf("want /metrics status code %d, have %d. Body:\n%s", want, have, b)
	}

	return nil
}

func runCommandAndTests(cmd *exec.Cmd, address string, fn func(pid int) error) error {
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	time.Sleep(50 * time.Millisecond)

	for i := range 10 {
		err := queryExporter(address)
		if err == nil {
			break
		}

		time.Sleep(500 * time.Millisecond)

		if cmd.Process == nil || i == 9 {
			return fmt.Errorf("can't start command %s %s", cmd.Stderr, cmd.Stdout)
		}
	}

	errc := make(chan error)

	go func(pid int) {
		errc <- fn(pid)
	}(cmd.Process.Pid)

	err = <-errc

	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	return err
}
