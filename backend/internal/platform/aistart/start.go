// Package aistart optionally launches the Python AI HTTP service as a child of the API process.
package aistart

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Options configures auto-start.
type Options struct {
	Enabled bool
	Python  string
	WorkDir string
	Host    string
	Port    int
	Logger  *slog.Logger
}

// Process is a running child AI server.
type Process struct {
	cmd *exec.Cmd
}

// Start launches `python -m swyngora_ai.serve` if Enabled and Python is set.
// Returns nil process when not started (not an error).
func Start(ctx context.Context, opts Options) (*Process, error) {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	if !opts.Enabled {
		log.Info("AI auto-start disabled (set AI_AUTOSTART=true and AI_PYTHON to enable)")
		return nil, nil
	}
	if opts.Python == "" {
		return nil, fmt.Errorf("AI_AUTOSTART=true requires AI_PYTHON (path to python with swyngora-ai installed)")
	}
	if opts.Host == "" {
		opts.Host = "127.0.0.1"
	}
	if opts.Port <= 0 {
		opts.Port = 8090
	}
	if portOpen(opts.Host, opts.Port) {
		log.Info("AI service already listening", "host", opts.Host, "port", opts.Port)
		return nil, nil
	}
	work := opts.WorkDir
	if work == "" {
		work = "ai"
	}
	abs, err := filepath.Abs(work)
	if err != nil {
		return nil, err
	}
	// If cwd is backend/, prefer sibling ai/ when work dir missing.
	if _, err := os.Stat(filepath.Join(abs, "src", "swyngora_ai")); err != nil {
		alt, _ := filepath.Abs(filepath.Join("..", "ai"))
		if _, err2 := os.Stat(filepath.Join(alt, "src", "swyngora_ai")); err2 == nil {
			abs = alt
		}
	}
	cmd := exec.CommandContext(ctx, opts.Python, "-m", "swyngora_ai.serve",
		"--host", opts.Host,
		"--port", strconv.Itoa(opts.Port),
	)
	cmd.Dir = abs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start AI: %w", err)
	}
	log.Info("AI service started", "pid", cmd.Process.Pid, "host", opts.Host, "port", opts.Port, "python", opts.Python)

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if portOpen(opts.Host, opts.Port) {
			return &Process{cmd: cmd}, nil
		}
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	log.Warn("AI process started but port not open yet", "port", opts.Port)
	return &Process{cmd: cmd}, nil
}

// Stop terminates the child if running.
func (p *Process) Stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_, _ = p.cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = p.cmd.Process.Kill()
	}
}

func portOpen(host string, port int) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	c, err := net.DialTimeout("tcp", addr, 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}
