package parser

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"feishu-codex-runner/internal/model"
)

var kvPattern = regexp.MustCompile(`#([a-zA-Z_]+)=(("[^"]+")|([^\s]+))`)

type ParseOptions struct {
	DefaultRepo    string
	DefaultTestCmd string
}

func ParseMessage(msg model.Message, opts ParseOptions) (model.Task, error) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return model.Task{}, errors.New("empty message")
	}
	task := model.Task{
		Repo:        opts.DefaultRepo,
		TestCmd:     opts.DefaultTestCmd,
		Mode:        "implement",
		Instruction: text,
		RequesterID: msg.SenderOpenID,
		ChatID:      msg.ChatID,
		MessageID:   msg.MessageID,
		ReceivedAt:  msg.CreateTime,
		RawText:     msg.Text,
	}

	if strings.HasPrefix(text, "{") {
		if err := parseJSON(text, &task); err == nil {
			return finalize(task)
		}
	}

	matches := kvPattern.FindAllStringSubmatch(text, -1)
	cleaned := text
	for _, m := range matches {
		k := strings.ToLower(m[1])
		v := strings.Trim(m[2], `"`)
		switch k {
		case "repo":
			task.Repo = v
		case "branch":
			task.Branch = v
		case "test", "test_cmd":
			task.TestCmd = v
		case "mode":
			task.Mode = v
		}
		cleaned = strings.Replace(cleaned, m[0], "", 1)
	}
	task.Instruction = strings.TrimSpace(cleaned)
	return finalize(task)
}

func parseJSON(text string, task *model.Task) error {
	var payload struct {
		Repo        string `json:"repo"`
		Branch      string `json:"branch"`
		TestCmd     string `json:"test_cmd"`
		Mode        string `json:"mode"`
		Task        string `json:"task"`
		Instruction string `json:"instruction"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return err
	}
	if payload.Repo != "" {
		task.Repo = payload.Repo
	}
	if payload.Branch != "" {
		task.Branch = payload.Branch
	}
	if payload.TestCmd != "" {
		task.TestCmd = payload.TestCmd
	}
	if payload.Mode != "" {
		task.Mode = payload.Mode
	}
	if payload.Task != "" {
		task.Instruction = payload.Task
	}
	if payload.Instruction != "" {
		task.Instruction = payload.Instruction
	}
	return nil
}

func finalize(task model.Task) (model.Task, error) {
	if task.Repo == "" {
		return model.Task{}, errors.New("repo is required")
	}
	if task.TestCmd == "" {
		task.TestCmd = "go test ./..."
	}
	if task.Mode == "" {
		task.Mode = "implement"
	}
	if task.Instruction == "" {
		return model.Task{}, errors.New("instruction is required")
	}
	return task, nil
}
