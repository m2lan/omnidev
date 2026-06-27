package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/omnidev/go-common/config"
)

// SandboxManager manages code execution in sandboxed environments.
type SandboxManager struct {
	cpuLimit    string
	memoryLimit string
	timeout     time.Duration
}

// NewSandboxManager creates a new sandbox manager.
func NewSandboxManager(cfg config.SandboxConfig) *SandboxManager {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	return &SandboxManager{
		cpuLimit:    cfg.CPULimit,
		memoryLimit: cfg.MemoryLimit,
		timeout:     timeout,
	}
}

// Execute runs code in a sandboxed environment.
func (s *SandboxManager) Execute(ctx context.Context, language, code string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var cmd *exec.Cmd

	switch language {
	case "python", "python3":
		cmd = exec.CommandContext(ctx, "python3", "-c", code)
	case "javascript", "js", "node":
		cmd = exec.CommandContext(ctx, "node", "-e", code)
	case "go":
		cmd = exec.CommandContext(ctx, "go", "run", "/dev/stdin")
		cmd.Stdin = bytes.NewReader([]byte(code))
	case "shell", "bash", "sh":
		cmd = exec.CommandContext(ctx, "sh", "-c", code)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Set resource limits
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/tmp",
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	latency := time.Since(start)

	result := map[string]interface{}{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"language":  language,
		"success":   err == nil,
		"latency_ms": latency.Milliseconds(),
	}

	if err != nil {
		result["error"] = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		}
	}

	return result, nil
}

// IsLanguageSupported checks if a language is supported.
func (s *SandboxManager) IsLanguageSupported(language string) bool {
	switch language {
	case "python", "python3", "javascript", "js", "node", "go", "shell", "bash", "sh":
		return true
	default:
		return false
	}
}

// SupportedLanguages returns a list of supported languages.
func (s *SandboxManager) SupportedLanguages() []string {
	return []string{"python", "javascript", "go", "shell"}
}
