package repo

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"feishu-codex-runner/internal/config"
)

type Manager struct {
	repos map[string]config.RepoConfig
}

func NewManager(items []config.RepoConfig) *Manager {
	m := &Manager{repos: make(map[string]config.RepoConfig, len(items))}
	for _, it := range items {
		m.repos[it.Name] = it
	}
	return m
}

func (m *Manager) Resolve(name string) (config.RepoConfig, error) {
	r, ok := m.repos[name]
	if !ok {
		return config.RepoConfig{}, fmt.Errorf("repo %s not found", name)
	}
	if !r.Allowed {
		return config.RepoConfig{}, fmt.Errorf("repo %s not allowed", name)
	}
	return r, nil
}

func EnsureCleanAndCheckout(ctx context.Context, repo config.RepoConfig, branch string) error {
	if err := ensureClean(ctx, repo.LocalPath); err != nil {
		return err
	}
	target := strings.TrimSpace(branch)
	if target == "" {
		target = strings.TrimSpace(repo.DefaultBranch)
	}
	if target == "" {
		return nil
	}
	_, err := runGit(ctx, repo.LocalPath, "checkout", target)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "did not match") || strings.Contains(err.Error(), "pathspec") {
		_, err = runGit(ctx, repo.LocalPath, "checkout", "-b", target)
	}
	return err
}

func ensureClean(ctx context.Context, path string) error {
	out, err := runGit(ctx, path, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("repository has uncommitted changes; refusing to run")
	}
	return nil
}

func runGit(ctx context.Context, workdir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func DiffStat(ctx context.Context, path string) string {
	out, err := runGit(ctx, path, "diff", "--stat")
	if err != nil {
		return "(failed to gather diff stat)"
	}
	if strings.TrimSpace(out) == "" {
		return "(no changes)"
	}
	return out
}

func DiffSnippet(ctx context.Context, path string, maxLines int) string {
	out, err := runGit(ctx, path, "diff")
	if err != nil {
		return "(failed to gather diff)"
	}
	lines := strings.Split(out, "\n")
	if len(lines) <= maxLines {
		return out
	}
	return strings.Join(lines[:maxLines], "\n") + "\n... (truncated)"
}
