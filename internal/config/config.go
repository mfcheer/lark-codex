package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type RepoConfig struct {
	Name          string
	LocalPath     string
	Allowed       bool
	DefaultBranch string
}

type Runtime struct {
	FeishuAppID      string
	FeishuAppSecret  string
	CodexBin         string
	PollInterval     time.Duration
	WorkDir          string
	ReposFile        string
	AllowListFile    string
	DefaultTestCmd   string
	ExecutionTimeout time.Duration
}

func LoadRuntime() (Runtime, error) {
	pollSec := readIntEnv("RUNNER_POLL_INTERVAL_SEC", 8)
	timeoutMin := readIntEnv("RUNNER_EXEC_TIMEOUT_MIN", 30)
	wd := os.Getenv("RUNNER_WORK_DIR")
	if wd == "" {
		wd = "./runner-data"
	}
	cfg := Runtime{
		FeishuAppID:      os.Getenv("FEISHU_APP_ID"),
		FeishuAppSecret:  os.Getenv("FEISHU_APP_SECRET"),
		CodexBin:         getenvDefault("CODEX_BIN", "codex"),
		PollInterval:     time.Duration(pollSec) * time.Second,
		WorkDir:          wd,
		ReposFile:        getenvDefault("RUNNER_REPOS_FILE", "./repos.yaml"),
		AllowListFile:    getenvDefault("RUNNER_ALLOWLIST_FILE", "./allowlist.yaml"),
		DefaultTestCmd:   getenvDefault("RUNNER_DEFAULT_TEST_CMD", "go test ./..."),
		ExecutionTimeout: time.Duration(timeoutMin) * time.Minute,
	}
	if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
		return Runtime{}, errors.New("FEISHU_APP_ID and FEISHU_APP_SECRET must be set")
	}
	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return Runtime{}, fmt.Errorf("create workdir: %w", err)
	}
	if absWD, err := filepath.Abs(cfg.WorkDir); err == nil {
		cfg.WorkDir = absWD
	}
	return cfg, nil
}

func LoadRepos(path string) ([]RepoConfig, error) {
	m, err := parseSimpleYAML(path)
	if err != nil {
		return nil, err
	}
	items, ok := m["repos"].([]map[string]string)
	if !ok {
		return nil, errors.New("repos.yaml must contain repos list")
	}
	out := make([]RepoConfig, 0, len(items))
	for _, it := range items {
		out = append(out, RepoConfig{
			Name:          it["name"],
			LocalPath:     it["local_path"],
			Allowed:       strings.EqualFold(it["allowed"], "true"),
			DefaultBranch: it["default_branch"],
		})
	}
	return out, nil
}

func LoadAllowList(path string) (map[string]struct{}, error) {
	m, err := parseSimpleYAML(path)
	if err != nil {
		return nil, err
	}
	vals, ok := m["open_ids"].([]string)
	if !ok {
		return nil, errors.New("allowlist.yaml must contain open_ids list")
	}
	set := make(map[string]struct{}, len(vals))
	for _, id := range vals {
		id = strings.TrimSpace(id)
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set, nil
}

// parseSimpleYAML supports just the subset used by repos.yaml and allowlist.yaml.
func parseSimpleYAML(path string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	root := map[string]any{}
	s := bufio.NewScanner(f)
	var currentKey string
	var currentObj map[string]string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "-") {
			currentKey = strings.TrimSuffix(line, ":")
			currentObj = nil
			continue
		}
		if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if strings.Contains(val, ":") {
				parts := strings.SplitN(val, ":", 2)
				currentObj = map[string]string{strings.TrimSpace(parts[0]): trimVal(parts[1])}
				list, _ := root[currentKey].([]map[string]string)
				list = append(list, currentObj)
				root[currentKey] = list
			} else {
				list, _ := root[currentKey].([]string)
				list = append(list, trimVal(val))
				root[currentKey] = list
			}
			continue
		}
		if strings.Contains(line, ":") && currentObj != nil {
			parts := strings.SplitN(line, ":", 2)
			currentObj[strings.TrimSpace(parts[0])] = trimVal(parts[1])
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return root, nil
}

func trimVal(v string) string {
	v = strings.TrimSpace(v)
	return strings.Trim(v, `"`)
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func readIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
