// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

// Package exectest allows for testing controlled responses from a an os/exec
// Cmd. It is imported and used in tests for any package that uses os/exec Cmds.
package exectest

import (
	"fmt"
	"os"
	"os/exec"
)

// GenFakeExecCommand works by running the specific test TestRunExecCmd in the
// package using the fake command as a separate process. That test should be a
// thin wrapper on RunExecCmd below.
// Adapted from https://npf.io/2015/06/testing-exec-command/.
func GenFakeExecCommand(output, err string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestRunExecCmd", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"BAGOUP_WANT_TEST_RUN_EXEC_CMD=1",
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT=%s", output),
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_ERROR=%s", err),
		}
		return cmd
	}
}

// RunExecCmd writes the messages specified as envrionment variables in
// GenFakeExecCommand, to stdout and stderr before exiting with the correct exit
// code.
func RunExecCmd() {
	if os.Getenv("BAGOUP_WANT_TEST_RUN_EXEC_CMD") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT"))
	err := os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_ERROR")
	if err != "" {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
