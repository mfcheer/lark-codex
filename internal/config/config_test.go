package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRepos(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "repos.yaml")
	_ = os.WriteFile(p, []byte("repos:\n  - name: aoi\n    local_path: /tmp/aoi\n    allowed: true\n    default_branch: main\n"), 0o644)
	repos, err := LoadRepos(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "aoi" || !repos[0].Allowed {
		t.Fatalf("unexpected repos: %+v", repos)
	}
}
