package report

import (
	"fmt"
	"strings"

	"feishu-codex-runner/internal/codex"
	"feishu-codex-runner/internal/model"
)

func Accepted(task model.Task) string {
	return fmt.Sprintf("✅ 任务已接收\ntask_id=%s\nrepo=%s branch=%s", task.ID, task.Repo, blankAs(task.Branch, "(default)"))
}

func Final(task model.Task, run codex.Result, diffStat, diffSnippet string) string {
	status := "✅ 成功"
	if run.ExitErr != nil || run.TestErr != nil {
		status = "❌ 失败"
	}
	if run.TimedOut {
		status = "❌ 超时"
	}
	parts := []string{status,
		fmt.Sprintf("task_id=%s", task.ID),
		fmt.Sprintf("耗时=%s", run.Duration.Round(1e9)),
		"\n[Codex 输出摘要]\n" + truncateLines(run.Output, 40),
		"\n[Diff Stat]\n" + truncateLines(diffStat, 30),
		"\n[Diff 摘要]\n" + truncateLines(diffSnippet, 60),
	}
	if run.TestOutput != "" {
		parts = append(parts, "\n[测试输出]\n"+truncateLines(run.TestOutput, 40))
	}
	if run.LogPath != "" {
		parts = append(parts, "\n完整日志: "+run.LogPath)
	}
	if run.ExitErr != nil {
		parts = append(parts, "\nCodex 执行错误: "+run.ExitErr.Error())
	}
	if run.TestErr != nil {
		parts = append(parts, "\n测试错误: "+run.TestErr.Error())
	}
	return strings.Join(parts, "\n")
}

func truncateLines(s string, max int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= max {
		return strings.TrimSpace(s)
	}
	return strings.Join(lines[:max], "\n") + "\n... (truncated)"
}

func blankAs(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}
