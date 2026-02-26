package model

import "time"

// Task is a parsed user instruction ready for execution.
type Task struct {
	ID           string
	Repo         string
	Branch       string
	TestCmd      string
	Mode         string
	Instruction  string
	RequesterID  string
	ChatID       string
	MessageID    string
	ReceivedAt   time.Time
	RawText      string
	ReplyMessage string
}

// Message represents a simplified Feishu message payload used by the runner.
type Message struct {
	MessageID    string
	ChatID       string
	SenderOpenID string
	Text         string
	CreateTime   time.Time
}
