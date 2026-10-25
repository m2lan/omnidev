package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/omnidev/go-common/config"
)

func TestMountSourceMapsWorkspaceToHostRoot(t *testing.T) {
	root := t.TempDir()
	hostRoot := t.TempDir()
	manager := NewSandboxManager(config.SandboxConfig{
		WorkspaceRoot:     root,
		HostWorkspaceRoot: hostRoot,
	})

	got, err := manager.mountSource(filepath.Join(root, "run-1"))
	if err != nil {
		t.Fatalf("mountSource() error = %v", err)
	}
	want := filepath.Join(hostRoot, "run-1")
	if got != want {
		t.Fatalf("mountSource() = %q, want %q", got, want)
	}
}

func TestMountSourceRejectsWorkspaceOutsideRoot(t *testing.T) {
	root := t.TempDir()
	manager := NewSandboxManager(config.SandboxConfig{
		WorkspaceRoot:     root,
		HostWorkspaceRoot: t.TempDir(),
	})

	if _, err := manager.mountSource(filepath.Join(root, "..", "outside")); err == nil {
		t.Fatal("mountSource() accepted a workspace outside the configured root")
	}
}

func TestLimitedBufferTruncatesWithoutBlockingWrites(t *testing.T) {
	buffer := &limitedBuffer{limit: 4}
	n, err := buffer.Write([]byte("abcdef"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 6 {
		t.Fatalf("Write() count = %d, want 6", n)
	}
	if got := buffer.String(); got != "abcd" {
		t.Fatalf("String() = %q, want %q", got, "abcd")
	}
	if !buffer.Truncated() {
		t.Fatal("Truncated() = false, want true")
	}
}

func TestSandboxExecuteDocker(t *testing.T) {
	if os.Getenv("SANDBOX_INTEGRATION_TEST") != "1" {
		t.Skip("set SANDBOX_INTEGRATION_TEST=1 to run the Docker sandbox smoke test")
	}

	manager := NewSandboxManager(config.SandboxConfig{
		WorkspaceRoot: t.TempDir(),
		ShellImage:    "alpine:3.19",
		Network:       "none",
		Timeout:       30 * time.Second,
	})
	workspace := manager.WorkspacePath("smoke")
	result, err := manager.Execute(
		context.Background(),
		workspace,
		"shell",
		"printf 'sandbox-ok' > result.txt && cat result.txt",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result["success"] != true {
		t.Fatalf("Execute() result = %#v", result)
	}
	if result["stdout"] != "sandbox-ok" {
		t.Fatalf("stdout = %q, want %q", result["stdout"], "sandbox-ok")
	}
	contents, err := os.ReadFile(filepath.Join(workspace, "result.txt"))
	if err != nil {
		t.Fatalf("read workspace result: %v", err)
	}
	if string(contents) != "sandbox-ok" {
		t.Fatalf("workspace result = %q, want %q", contents, "sandbox-ok")
	}
}
