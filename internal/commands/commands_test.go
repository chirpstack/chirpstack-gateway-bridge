package commands

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestParseCommandLine(t *testing.T) {
	assert := require.New(t)

	tests := []struct {
		In    string
		Out   []string
		Error error
	}{
		{
			In:  "/path/to/bin arg1 arg2 arg3",
			Out: []string{"/path/to/bin", "arg1", "arg2", "arg3"},
		},
	}

	for _, tst := range tests {
		out, err := ParseCommandLine(tst.In)
		assert.Equal(tst.Error, err)
		if err != nil {
			continue
		}
		assert.Equal(tst.Out, out)
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		Name     string
		Commands map[string]command

		Command     string
		Stdin       []byte
		Environment map[string]string

		ExpectedStdout []byte
		ExpectedStdErr []byte
		ExpectedError  error
	}{
		{
			Name:          "command not configured",
			Command:       "reboot",
			ExpectedError: errors.New("command does not exist"),
		},
		{
			Name: "word count stdin",
			Commands: map[string]command{
				"wordcount": command{
					Command:              "wc -w",
					MaxExecutionDuration: time.Second,
				},
			},
			Command:        "wordcount",
			Stdin:          []byte("foo bar test bar"),
			ExpectedStdout: []byte("4\n"),
			ExpectedStdErr: []byte{},
		},
		{
			Name: "execution time epxired",
			Commands: map[string]command{
				"sleep": command{
					Command:              "sleep 1",
					MaxExecutionDuration: time.Millisecond,
				},
			},
			Command:       "sleep",
			ExpectedError: errors.New("waiting for command to finish error: signal: killed"),
		},
		{
			Name: "environment variables",
			Commands: map[string]command{
				"printenv": command{
					Command:              "printenv FOO",
					MaxExecutionDuration: time.Second,
				},
			},
			Command: "printenv",
			Environment: map[string]string{
				"FOO": "bar",
			},
			ExpectedStdout: []byte("bar\n"),
			ExpectedStdErr: []byte{},
		},
		{
			Name: "stdout and stderr",
			Commands: map[string]command{
				"echo": command{
					Command:              `sh -c 'echo "foo" >&1; echo "bar" >&2'`,
					MaxExecutionDuration: time.Second,
				},
			},
			Command:        "echo",
			ExpectedStdout: []byte("foo\n"),
			ExpectedStdErr: []byte("bar\n"),
		},
		{
			Name: "executable not found",
			Commands: map[string]command{
				"foobar": command{
					Command:              "foobartest",
					MaxExecutionDuration: time.Second,
				},
			},
			Command:       "foobar",
			ExpectedError: errors.New(`starting command error: exec: "foobartest": executable file not found in $PATH`),
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			commands = tst.Commands

			stdout, stderr, err := execute(tst.Command, tst.Stdin, tst.Environment)
			if tst.ExpectedError != nil && err != nil {
				assert.Equal(tst.ExpectedError.Error(), err.Error())
			} else {
				assert.Equal(tst.ExpectedError, err)
			}
			assert.Equal(tst.ExpectedStdout, stdout)
			assert.Equal(tst.ExpectedStdErr, stderr)
		})
	}
}
