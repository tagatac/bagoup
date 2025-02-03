// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package exectest provides a method for mocking exec.Command.
// Adapted from https://npf.io/2015/06/testing-exec-command/.
package exectest

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

const (
	_envvarRunTestFlag = "EXECTEST_WANT_TEST_RUN_EXEC_CMD"
	_envvarOutput      = "EXECTEST_OUTPUT"
	_envvarError       = "EXECTEST_ERROR"
	_envvarExitCode    = "EXECTEST_EXITCODE"
	_envvarGoCoverDir  = "GOCOVERDIR"
)

// GenFakeExecCommand generates a mock of exec.Command. When used to create and run an exec.Cmd,
// that Cmd runs the specified test in the package which created the mock as a separate
// process.
//
// **testname** should be the name of a thin wrapper on RunExecCmd below. For example:
//
//	package app
//
//	func TestApp(t *testing.T) {
//	  ...
//	  mockCommand := exectest.GenFakeExecCommand("TestRunExecCmd", "mock output", "mock error", 1)
//	  mockCmd := mockCommand.Cmd("/opt/app/bin/app")
//	  ...
//	  mockCmd.Run()
//	  ...
//	}
//
//	func TestRunExecCmd(t *testing.T) { exectest.RunExecCmd() }
//
// **output** is the output to be emitted to stdout by the mock Cmd when run.
// **err** is the output to be emitted to stderr by the mock Cmd when run.
// **exitCode** is the exit code to be returned by the mock Cmd when run.
func GenFakeExecCommand(testname, output, err string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{fmt.Sprintf("-test.run=%s", testname), "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			fmt.Sprintf("%s=1", _envvarRunTestFlag),
			fmt.Sprintf("%s=%s", _envvarOutput, output),
			fmt.Sprintf("%s=%s", _envvarError, err),
			fmt.Sprintf("%s=%s", _envvarExitCode, strconv.Itoa(exitCode)),
			fmt.Sprintf("%s=%s", _envvarGoCoverDir, os.TempDir()),
		}
		return cmd
	}
}

// RunExecCmd writes the messages specified as environment variables in
// GenFakeExecCommand, to stdout and stderr before exiting with the correct exit
// code.
func RunExecCmd() {
	if os.Getenv(_envvarRunTestFlag) != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv(_envvarOutput))
	err := os.Getenv(_envvarError)
	if err != "" {
		fmt.Fprint(os.Stderr, err)
	}
	exitCodeStr := os.Getenv(_envvarExitCode)
	exitCode, convErr := strconv.Atoi(exitCodeStr)
	if convErr != nil {
		panic(fmt.Sprintf("failed to convert exit code %q to an int", exitCodeStr))
	}
	os.Exit(exitCode)
}
