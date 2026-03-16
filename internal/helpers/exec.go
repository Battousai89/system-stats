package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"unicode/utf16"

	"system-stats/internal/config"
)

// RunCommandWithTimeout executes a command with timeout
func RunCommandWithTimeout(name string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.Output()
}

// RunPowerShellCommand executes a PowerShell command with proper UTF-16LE output handling
func RunPowerShellCommand(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-NonInteractive",
		"-Command", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.Bytes()

	// PowerShell outputs UTF-16LE with BOM
	if len(output) >= 2 && output[0] == 0xFF && output[1] == 0xFE {
		// UTF-16LE BOM found, convert
		utf16Data := make([]uint16, (len(output)-2)/2)
		for i := 2; i < len(output); i += 2 {
			if i+1 < len(output) {
				utf16Data[(i-2)/2] = uint16(output[i]) | uint16(output[i+1])<<8
			}
		}
		return []byte(string(utf16.Decode(utf16Data))), err
	}

	// UTF-8 BOM
	if len(output) >= 3 && output[0] == 0xEF && output[1] == 0xBB && output[2] == 0xBF {
		return output[3:], err
	}

	// Without BOM - assume UTF-8
	return output, err
}

// RunShellCommand executes a shell command (bash/sh) on Unix-like systems
func RunShellCommand(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	var shell string
	var arg string

	// Determine shell based on OS
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
		arg = "/C"
	} else {
		shell = "/bin/sh"
		arg = "-c"
	}

	cmd := exec.CommandContext(ctx, shell, arg, script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.Bytes(), err
}

// RunBashCommand executes a bash command on Unix-like systems
func RunBashCommand(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.Bytes(), err
}

// ParseJSON parses a JSON string
func ParseJSON(data string, v interface{}) error {
	if data == "" || data == "null" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}
