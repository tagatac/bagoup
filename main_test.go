package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Masterminds/semver"

	"github.com/stretchr/testify/assert"
)

func TestGetMacOSVersion(t *testing.T) {
	tests := []struct {
		msg          string
		swVersOutput string
		swVersErr    string
		wantVersion  *semver.Version
		wantErr      string
	}{
		{
			msg:          "catalina",
			swVersOutput: "10.15.3\n",
			wantVersion:  semver.MustParse("10.15.3"),
		},
		{
			msg:       "sw_vers error",
			swVersErr: "beep boop beep\n",
			wantErr:   "call sw_vers: exit status 1",
		},
		{
			msg:          "bad version",
			swVersOutput: "asdf",
			wantErr:      "parse semantic version \"asdf\": Invalid Semantic Version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			v, err := getMacOSVersion(genFakeExecCommand(tt.swVersOutput, tt.swVersErr))
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, v.Equal(tt.wantVersion), "Expected: %s\nGot: %s\n", tt.wantVersion, v)
		})
	}
}

func genFakeExecCommand(output, err string) func(string, ...string) *exec.Cmd {
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

func TestRunExecCmd(t *testing.T) {
	if os.Getenv("BAGOUP_WANT_TEST_RUN_EXEC_CMD") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT"))
	err := os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_ERROR")
	if err != "" {
		fmt.Fprintf(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		msg   string
		dirty string
		clean string
	}{
		{
			msg:   "already clean",
			dirty: "+14155555555",
			clean: "+14155555555",
		},
		{
			msg:   "dirty",
			dirty: "+1 (415) 555-5555",
			clean: "+14155555555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.clean, sanitizePhone(tt.dirty))
		})
	}
}
