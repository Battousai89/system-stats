package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"unicode/utf16"

	"system-stats/internal/config"
)

// RunCommandWithTimeout выполняет команду с таймаутом
func RunCommandWithTimeout(name string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.Output()
}

// RunPowerShellCommand выполняет PowerShell команду с правильной обработкой UTF-16LE вывода
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

	// PowerShell выводит UTF-16LE с BOM
	if len(output) >= 2 && output[0] == 0xFF && output[1] == 0xFE {
		// UTF-16LE BOM найден, конвертируем
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

	// Без BOM - предполагаем UTF-8
	return output, err
}

// ParseJSON парсит JSON строку
func ParseJSON(data string, v interface{}) error {
	if data == "" || data == "null" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}
