package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/omnidev/go-common/config"
)

const (
	defaultWorkspaceRoot = "./data/agent-workspaces"
	maxOutputBytes       = 1 << 20
	defaultTimeout       = 120 * time.Second
)

// SandboxManager runs agent code inside throwaway Docker containers. Each run
// gets its own host directory mounted at /workspace; containers have no
// network, a read-only root filesystem and CPU/memory/PID limits.
type SandboxManager struct {
	cfg config.SandboxConfig
}

// NewSandboxManager creates a sandbox manager from configuration.
func NewSandboxManager(cfg config.SandboxConfig) *SandboxManager {
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = defaultWorkspaceRoot
	}
	if cfg.Network == "" {
		cfg.Network = "none"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.CPULimit == "" {
		cfg.CPULimit = "1"
	}
	if cfg.MemoryLimit == "" {
		cfg.MemoryLimit = "512m"
	}
	if cfg.PythonImage == "" {
		cfg.PythonImage = "python:3.12-alpine"
	}
	if cfg.NodeImage == "" {
		cfg.NodeImage = "node:22-alpine"
	}
	if cfg.GoImage == "" {
		cfg.GoImage = "golang:1.22-alpine"
	}
	if cfg.ShellImage == "" {
		cfg.ShellImage = "alpine:3.19"
	}
	return &SandboxManager{cfg: cfg}
}

// WorkspaceRoot returns the directory that stores per-run workspaces.
func (s *SandboxManager) WorkspaceRoot() string { return s.cfg.WorkspaceRoot }

// WorkspacePath returns the host directory for a run.
func (s *SandboxManager) WorkspacePath(runID string) string {
	return filepath.Join(s.cfg.WorkspaceRoot, runID)
}

// Execute runs code for the given language inside a Docker container.
func (s *SandboxManager) Execute(ctx context.Context, workspace, language, code string) (map[string]interface{}, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("code is required")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker sandbox unavailable: %w", err)
	}

	image, argv, err := s.imageCommand(language)
	if err != nil {
		return nil, err
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	if err := os.MkdirAll(absWorkspace, 0o750); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	mountSource, err := s.mountSource(absWorkspace)
	if err != nil {
		return nil, err
	}

	container := fmt.Sprintf("omnidev-agent-%d-%d", os.Getpid(), time.Now().UnixNano())
	args := []string{
		"run", "--rm", "-i",
		"--name", container,
		"--network", s.cfg.Network,
		"--memory", s.cfg.MemoryLimit,
		"--cpus", s.cfg.CPULimit,
		"--pids-limit", "128",
		"--security-opt", "no-new-privileges",
		"--cap-drop", "ALL",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"-v", mountSource + ":/workspace:rw",
		"-w", "/workspace",
		"-e", "HOME=/tmp",
		"-e", "GOCACHE=/tmp/go-cache",
		"-e", "GOMODCACHE=/tmp/go-mod",
		"-e", "GOPATH=/tmp/go",
		"-e", "NPM_CONFIG_CACHE=/tmp/npm",
		"-e", "PYTHONDONTWRITEBYTECODE=1",
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		image,
	}
	args = append(args, argv...)

	execCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "docker", args...)
	cmd.Stdin = strings.NewReader(code)
	stdout := &limitedBuffer{limit: maxOutputBytes}
	stderr := &limitedBuffer{limit: maxOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	runErr := cmd.Run()
	latency := time.Since(start)

	// Killing the docker CLI does not always stop the container.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = exec.CommandContext(cleanupCtx, "docker", "rm", "-f", container).Run()
	cleanupCancel()

	result := map[string]interface{}{
		"stdout":     stdout.String(),
		"stderr":     stderr.String(),
		"language":   language,
		"sandbox":    "docker",
		"image":      image,
		"success":    runErr == nil,
		"latency_ms": latency.Milliseconds(),
	}
	if stdout.Truncated() || stderr.Truncated() {
		result["truncated"] = true
	}
	if runErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result["error"] = fmt.Sprintf("sandbox timed out after %s", s.cfg.Timeout)
		} else {
			result["error"] = runErr.Error()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		}
	}

	return result, nil
}

// mountSource maps the gateway-local workspace to the path the Docker daemon
// must bind. They differ when the gateway runs inside a container itself.
func (s *SandboxManager) mountSource(absWorkspace string) (string, error) {
	if s.cfg.HostWorkspaceRoot == "" {
		return absWorkspace, nil
	}
	absRoot, err := filepath.Abs(s.cfg.WorkspaceRoot)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absWorkspace)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("workspace %s is outside sandbox root %s", absWorkspace, absRoot)
	}
	return filepath.Join(s.cfg.HostWorkspaceRoot, rel), nil
}

func (s *SandboxManager) imageCommand(language string) (string, []string, error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "python3":
		return s.cfg.PythonImage, []string{"python3", "-"}, nil
	case "javascript", "js", "node":
		return s.cfg.NodeImage, []string{"node", "-"}, nil
	case "go":
		return s.cfg.GoImage, []string{"sh", "-c", "cat > /tmp/main.go && go run /tmp/main.go"}, nil
	case "shell", "bash", "sh":
		return s.cfg.ShellImage, []string{"sh", "-s"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// IsLanguageSupported reports whether the sandbox can execute a language.
func (s *SandboxManager) IsLanguageSupported(language string) bool {
	_, _, err := s.imageCommand(language)
	return err == nil
}

// SupportedLanguages returns the languages available in the sandbox.
func (s *SandboxManager) SupportedLanguages() []string {
	return []string{"python", "javascript", "go", "shell"}
}

type limitedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *limitedBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}
