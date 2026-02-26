package orchestrator

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"feishu-codex-runner/internal/codex"
	"feishu-codex-runner/internal/config"
	"feishu-codex-runner/internal/feishu"
	"feishu-codex-runner/internal/model"
	"feishu-codex-runner/internal/parser"
	"feishu-codex-runner/internal/repo"
	"feishu-codex-runner/internal/report"
	"feishu-codex-runner/internal/store"
)

type App struct {
	cfg       config.Runtime
	feishu    *feishu.Client
	repoMgr   *repo.Manager
	allowList map[string]struct{}
	store     *store.JSONStore
	codex     codex.Runner
	state     store.State
	parseOpts parser.ParseOptions
}

func New(cfg config.Runtime, repos []config.RepoConfig, allow map[string]struct{}) (*App, error) {
	st := store.NewJSONStore(filepath.Join(cfg.WorkDir, "state.json"))
	state, err := st.Load()
	if err != nil {
		return nil, err
	}
	return &App{
		cfg:       cfg,
		feishu:    feishu.NewClient(cfg.FeishuAppID, cfg.FeishuAppSecret),
		repoMgr:   repo.NewManager(repos),
		allowList: allow,
		store:     st,
		state:     state,
		codex:     codex.Runner{Bin: cfg.CodexBin, WorkDir: filepath.Join(cfg.WorkDir, "logs"), Timeout: cfg.ExecutionTimeout, MaxOutput: 12000},
		parseOpts: parser.ParseOptions{DefaultTestCmd: cfg.DefaultTestCmd},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()
	if err := a.pollOnce(ctx); err != nil {
		log.Printf("poll error: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.pollOnce(ctx); err != nil {
				log.Printf("poll error: %v", err)
			}
		}
	}
}

func (a *App) pollOnce(ctx context.Context) error {
	start := time.Unix(a.state.LastPollUnix, 0)
	if a.state.LastPollUnix == 0 {
		start = time.Now().Add(-30 * time.Minute)
	}
	cursor := a.state.Cursor
	for {
		msgs, nextCursor, err := a.feishu.FetchMessages(ctx, start, cursor)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			if _, seen := a.state.Processed[msg.MessageID]; seen {
				continue
			}
			if err := a.handleMessage(ctx, msg); err != nil {
				log.Printf("handle message %s failed: %v", msg.MessageID, err)
				continue
			}
			a.state.Processed[msg.MessageID] = time.Now().Unix()
		}
		cursor = nextCursor
		if cursor == "" {
			break
		}
	}
	a.state.Cursor = cursor
	a.state.LastPollUnix = time.Now().Unix()
	return a.store.Save(a.state)
}

func (a *App) handleMessage(ctx context.Context, msg model.Message) error {
	if _, ok := a.allowList[msg.SenderOpenID]; !ok {
		return a.feishu.SendText(ctx, msg.ChatID, "⛔ 无权限触发 runner")
	}
	task, err := parser.ParseMessage(msg, a.parseOpts)
	if err != nil {
		return a.feishu.SendText(ctx, msg.ChatID, "⚠️ 指令解析失败: "+err.Error())
	}
	task.ID = makeTaskID(msg.MessageID)
	if err := codex.ValidateSafety(task.Instruction, task.TestCmd); err != nil {
		return a.feishu.SendText(ctx, msg.ChatID, "⛔ 任务被拒绝: "+err.Error())
	}
	if err := a.feishu.SendText(ctx, msg.ChatID, report.Accepted(task)); err != nil {
		return err
	}

	rc, err := a.repoMgr.Resolve(task.Repo)
	if err != nil {
		return a.feishu.SendText(ctx, msg.ChatID, "⛔ Repo 校验失败: "+err.Error())
	}
	if err := repo.EnsureCleanAndCheckout(ctx, rc, task.Branch); err != nil {
		return a.feishu.SendText(ctx, msg.ChatID, "⛔ Repo 状态不满足执行条件: "+err.Error())
	}

	run := a.codex.Execute(ctx, task, rc.LocalPath)
	if run.ExitErr == nil && !run.TimedOut {
		tout, terr := a.codex.RunTests(ctx, task, rc.LocalPath)
		run.TestOutput, run.TestErr = tout, terr
	}
	ds := repo.DiffStat(ctx, rc.LocalPath)
	diff := repo.DiffSnippet(ctx, rc.LocalPath, 120)
	return a.feishu.SendText(ctx, msg.ChatID, report.Final(task, run, ds, diff))
}

func makeTaskID(seed string) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s:%d", seed, time.Now().UnixNano())))
	return hex.EncodeToString(h[:])[:12]
}
