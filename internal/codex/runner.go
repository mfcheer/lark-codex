package codex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"feishu-codex-runner/internal/model"
)

type Runner struct {
	Bin       string
	WorkDir   string
	Timeout   time.Duration
	MaxOutput int
}

type Result struct {
	Prompt     string
	Output     string
	LogPath    string
	Duration   time.Duration
	TimedOut   bool
	ExitErr    error
	TestOutput string
	TestErr    error
}

var blockedKeywords = []string{"rm -rf", "git push --force", "sudo ", "mkfs", "shutdown", "reboot"}

func ValidateSafety(parts ...string) error {
	for _, part := range parts {
		lower := strings.ToLower(part)
		for _, kw := range blockedKeywords {
			if strings.Contains(lower, kw) {
				return fmt.Errorf("instruction rejected due to dangerous keyword: %s", kw)
			}
		}
	}
	return nil
}

func (r Runner) Execute(ctx context.Context, task model.Task, repoPath string) Result {
	start := time.Now()
	result := Result{}
	prompt := buildPrompt(task)
	result.Prompt = prompt

	if err := os.MkdirAll(r.WorkDir, 0o755); err != nil {
		result.ExitErr = err
		return result
	}
	logPath := filepath.Join(r.WorkDir, fmt.Sprintf("task-%s.log", task.ID))
	result.LogPath = logPath

	cctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, r.Bin, "exec", "-")
	cmd.Dir = repoPath
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	if cctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
	}
	result.Output = trim(string(out), r.MaxOutput)
	_ = os.WriteFile(logPath, out, 0o644)
	result.ExitErr = err
	result.Duration = time.Since(start)
	return result
}

func (r Runner) RunTests(ctx context.Context, task model.Task, repoPath string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, "bash", "-lc", task.TestCmd)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	return trim(string(out), r.MaxOutput), err
}

func buildPrompt(task model.Task) string {
	return fmt.Sprintf(`你正在一个 Go 项目仓库中工作。
只做完成任务所需的最小改动。
不要做大规模重构，除非任务要求。
修改后必须运行测试：%s
输出：
1. 改动摘要（要点）
2. 涉及文件列表
3. 如何验证（包含测试命令与结果）
4. 若失败，给出下一步建议

任务模式：%s
用户任务：%s
`, task.TestCmd, task.Mode, task.Instruction)
}

func trim(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated)"
}
