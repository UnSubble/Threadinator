package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/unsubble/threadinator/internal/executor"
)

func TestParseArgs(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		expectedConfig *executor.Config
		expectedErr    bool
	}{
		{
			name: "Valid single command",
			args: []string{"-e", "ls -l"},
			expectedConfig: &executor.Config{
				Commands: []executor.Command{
					{
						Command: "ls",
						Args:    []string{"-l"},
						Times:   1,
					},
				},
				ThreadCount: 5,
				Timeout:     10 * time.Second,
				UsePipeline: false,
				Verbose:     false,
			},
			expectedErr: false,
		},
		{
			name: "Multiple commands with semicolon",
			args: []string{"-c", "7", "-e", "ls:6; pwd"},
			expectedConfig: &executor.Config{
				Commands: []executor.Command{
					{
						Command: "ls",
						Args:    []string{},
						Times:   6,
					},
					{
						Command: "pwd",
						Args:    []string{},
						Times:   1,
					},
				},
				ThreadCount: 7,
				Timeout:     10 * time.Second,
				UsePipeline: false,
				Verbose:     false,
			},
			expectedErr: false,
		},
		{
			name: "Custom thread count and timeout",
			args: []string{"-e", "test", "-c", "10", "-t", "30"},
			expectedConfig: &executor.Config{
				Commands: []executor.Command{
					{
						Command: "test",
						Args:    []string{},
						Times:   1,
					},
				},
				ThreadCount: 10,
				Timeout:     30 * time.Second,
				UsePipeline: false,
				Verbose:     false,
			},
			expectedErr: false,
		},
		{
			name:           "Missing command",
			args:           []string{},
			expectedConfig: nil,
			expectedErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := ParseArgs(tc.args)

			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedConfig.Commands, config.Commands)
			assert.Equal(t, tc.expectedConfig.ThreadCount, config.ThreadCount)
			assert.Equal(t, tc.expectedConfig.Timeout, config.Timeout)
			assert.Equal(t, tc.expectedConfig.UsePipeline, config.UsePipeline)
			assert.Equal(t, tc.expectedConfig.Verbose, config.Verbose)
		})
	}
}

func TestParseCommands(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedCmds []executor.Command
	}{
		{
			name:  "Simple single command",
			input: "ls -l",
			expectedCmds: []executor.Command{
				{
					Command: "ls",
					Args:    []string{"-l"},
					Times:   1,
				},
			},
		},
		{
			name:  "Multiple commands with semicolon",
			input: "ls -l; pwd",
			expectedCmds: []executor.Command{
				{
					Command: "ls",
					Args:    []string{"-l"},
					Times:   1,
				},
				{
					Command: "pwd",
					Args:    []string{},
					Times:   1,
				},
			},
		},
		{
			name:  "Quoted commands",
			input: "'ls -l'; \"pwd\"",
			expectedCmds: []executor.Command{
				{
					Command: "ls",
					Args:    []string{"-l"},
					Times:   1,
				},
				{
					Command: "pwd",
					Args:    []string{},
					Times:   1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmds := parseCommands(tc.input)
			assert.Equal(t, tc.expectedCmds, cmds)
		})
	}
}

func TestSplitCommand(t *testing.T) {
	testCases := []struct {
		input       string
		expectedCmd executor.Command
	}{
		{
			input: "ls -l",
			expectedCmd: executor.Command{
				Command: "ls",
				Args:    []string{"-l"},
				Times:   1,
			},
		},
		{
			input: "echo hello world",
			expectedCmd: executor.Command{
				Command: "echo",
				Args:    []string{"hello", "world"},
				Times:   1,
			},
		},
		{
			input: "single",
			expectedCmd: executor.Command{
				Command: "single",
				Args:    []string{},
				Times:   1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			cmd := parseCommands(tc.input)
			assert.Equal(t, 1, len(cmd))
			assert.Equal(t, tc.expectedCmd, cmd[0])
		})
	}
}
