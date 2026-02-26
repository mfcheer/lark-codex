package parser

import (
	"testing"
	"time"

	"feishu-codex-runner/internal/model"
)

func TestParseMessageFlags(t *testing.T) {
	msg := model.Message{MessageID: "m1", ChatID: "c1", SenderOpenID: "u1", CreateTime: time.Now(), Text: `#repo=aoi-service #branch=feat/jwt #test_cmd="go test ./..." add jwt middleware`}
	task, err := ParseMessage(msg, ParseOptions{DefaultTestCmd: "go test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	if task.Repo != "aoi-service" || task.Branch != "feat/jwt" {
		t.Fatalf("unexpected parsed task: %+v", task)
	}
}

func TestParseMessageJSON(t *testing.T) {
	msg := model.Message{MessageID: "m1", ChatID: "c1", SenderOpenID: "u1", CreateTime: time.Now(), Text: `{"repo":"aoi-service","task":"fix healthz","test_cmd":"go test ./..."}`}
	task, err := ParseMessage(msg, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if task.Instruction != "fix healthz" || task.Repo != "aoi-service" {
		t.Fatalf("unexpected task: %+v", task)
	}
}
