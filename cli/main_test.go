package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGetMonthName(t *testing.T) {
	months := []string{"Янв", "Фев", "Март", "Апр", "Май", "Июнь", "Июль", "Авг", "Сен", "Окт", "Ноя", "Дек"}

	testCases := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"Янв", "Янв", false},
		{"1", "Янв", false},
		{"12", "Дек", false},
		{"янв", "Янв", false},
		{"март", "Март", false},
		{"invalid", "", true},
		{"13", "", true},
		{"", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, err := getMonthName(tc.input, months)
			if (err != nil) != tc.hasError {
				t.Errorf("getMonthName(%q): expected error %v, got %v", tc.input, tc.hasError, err)
			}
			if actual != tc.expected {
				t.Errorf("getMonthName(%q): expected %q, got %q", tc.input, tc.expected, actual)
			}
		})
	}
}

func TestMainCLI(t *testing.T) {
	// Build the CLI binary
	cmd := exec.Command("go", "build", "-o", "../test_calendar_solver_cli", ".")
	buildOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, buildOutput)
	}
	defer os.Remove("../test_calendar_solver_cli")

	testCases := []struct {
		name          string
		args          []string
		expectedOut   string
		expectErr     bool
		notExpectedOut string
	}{
		{
			name:        "Specific Date",
			args:        []string{"--day", "1", "--month", "1"},
			expectedOut: "Command line date: 1 Янв",
		},
		{
			name:        "Test Only",
			args:        []string{"--test-only"},
			expectedOut: "TESTING OTHER DATES:",
			notExpectedOut: "Solving calendar board for:",
		},
		{
			name:        "Invalid Month",
			args:        []string{"--day", "1", "--month", "invalid"},
			expectedOut: "Error: invalid month: invalid",
			expectErr:   true,
		},
		{
			name:        "No args",
			args:        []string{},
			expectedOut: "Using current date:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("../test_calendar_solver_cli", tc.args...)
			output, err := cmd.CombinedOutput()

			if (err != nil) != tc.expectErr {
				t.Logf("Output: %s", output)
				t.Fatalf("Expected error: %v, got: %v", tc.expectErr, err)
			}

			if !strings.Contains(string(output), tc.expectedOut) {
				t.Errorf("Expected output to contain %q, but it didn't. Output: %s", tc.expectedOut, output)
			}

			if tc.notExpectedOut != "" && strings.Contains(string(output), tc.notExpectedOut) {
				t.Errorf("Expected output not to contain %q, but it did. Output: %s", tc.notExpectedOut, output)
			}
		})
	}
}