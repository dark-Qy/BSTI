package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Executor struct {
	bin        string
	sessionDir string
	timeout    time.Duration
	extraEnv   []string
}

type Output struct {
	Stdout string
	Stderr string
}

func NewExecutor(bin, sessionDir string, timeout time.Duration) *Executor {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Executor{bin: bin, sessionDir: sessionDir, timeout: timeout}
}

func (e *Executor) WithEnv(env ...string) *Executor {
	e.extraEnv = append(e.extraEnv, env...)
	return e
}

func (e *Executor) ConfigDir() string {
	return filepath.Join(e.sessionDir, "lark-cli")
}

func (e *Executor) Validate(args []string) error {
	if len(args) == 0 {
		return errors.New("empty lark-cli command")
	}
	switch args[0] {
	case "config":
		return validateConfig(args)
	case "auth":
		return validateAuth(args)
	case "im":
		return validateIM(args)
	case "docs":
		return requireShortcut(args, "+search", "+fetch")
	case "calendar":
		return requireShortcut(args, "+agenda")
	case "task":
		return requireShortcut(args, "+get-my-tasks")
	case "mail":
		return requireShortcut(args, "+triage", "+message")
	case "vc":
		return requireShortcut(args, "+search", "+notes")
	case "contact":
		return requireShortcut(args, "+get-user")
	default:
		return fmt.Errorf("lark-cli command %q is not allowed", args[0])
	}
}

func validateIM(args []string) error {
	if len(args) < 2 {
		return errors.New("im shortcut is required")
	}
	if err := requireShortcut(args, "+messages-search", "+chat-messages-list", "+chat-search"); err == nil {
		return nil
	}
	if len(args) >= 3 {
		switch {
		case args[1] == "chat.members" && args[2] == "get":
			return nil
		case args[1] == "chats" && args[2] == "get":
			return nil
		}
	}
	return fmt.Errorf("im shortcut %q is not allowed", args[1])
}

func validateConfig(args []string) error {
	if len(args) == 3 && args[1] == "init" && args[2] == "--new" {
		return nil
	}
	return errors.New("only config init --new is allowed")
}

func (e *Executor) Run(ctx context.Context, args []string) (Output, error) {
	if err := e.Validate(args); err != nil {
		return Output{}, err
	}
	if err := os.MkdirAll(e.ConfigDir(), 0700); err != nil {
		return Output{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, e.bin, args...)
	cmd.Dir = e.sessionDir
	cmd.Env = append(os.Environ(), e.extraEnv...)
	cmd.Env = append(cmd.Env, "LARKSUITE_CLI_CONFIG_DIR="+e.ConfigDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := Output{Stdout: stdout.String(), Stderr: stderr.String()}
	if runCtx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("lark-cli command timed out")
	}
	if err != nil {
		return out, fmt.Errorf("lark-cli command failed: %w: %s", err, stderr.String())
	}
	return out, nil
}

func validateAuth(args []string) error {
	if len(args) < 2 {
		return errors.New("auth subcommand is required")
	}
	if args[1] == "status" {
		return nil
	}
	if args[1] != "login" {
		return fmt.Errorf("auth subcommand %q is not allowed", args[1])
	}
	if contains(args, "--device-code") {
		return nil
	}
	if contains(args, "--no-wait") && contains(args, "--scope") {
		return nil
	}
	return errors.New("auth login must use --no-wait with --scope or --device-code")
}

func requireShortcut(args []string, allowed ...string) error {
	if len(args) < 2 {
		return fmt.Errorf("%s shortcut is required", args[0])
	}
	for _, item := range allowed {
		if args[1] == item {
			return nil
		}
	}
	return fmt.Errorf("%s shortcut %q is not allowed", args[0], args[1])
}

func contains(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}
