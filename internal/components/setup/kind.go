package setup

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type ExecResult struct {
	Command []string
	Error   error
	Stdout  string
	StdErr  string
}

const (
	KIND        = "kind"
	KINDCOMMAND = "kind"
)

var (
	// kind cluster create config
	kindConfigFile string
)

// setup for kind, invoke from command line
func KindSetupInCommand() {
	kindConfigFile = flags.File

	execResult := createKindCluster()
	err := execResult.Error
	if err != nil {
		cmd := strings.Join(execResult.Command, " ")
		logger.Log.Errorf("Kind cluster create exited abnormally whilst running [%s]\n"+
			"err: %s\nstdout: %s\nstderr: %s", cmd, err, execResult.Stdout, execResult.StdErr)
	} else {
		defer cleanupKindCluster()
	}
}

func kindExec(args []string) ExecResult {
	cmd := exec.Command(KINDCOMMAND, args...)
	var stdoutBytes, stderrBytes bytes.Buffer
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes

	err := cmd.Run()
	execCmd := []string{KINDCOMMAND}
	execCmd = append(execCmd, args...)

	return ExecResult{
		Command: execCmd,
		Error:   err,
		Stdout:  stdoutBytes.String(),
		StdErr:  stderrBytes.String(),
	}
}

func createKindCluster() ExecResult {
	args := []string{"create", "cluster", "--config", kindConfigFile}

	logger.Log.Info("creating kind cluster...")
	return kindExec(args)
}

func cleanupKindCluster() ExecResult {
	args := []string{"delete", "cluster"}

	logger.Log.Info("deleting kind cluster...")
	return kindExec(args)
}
