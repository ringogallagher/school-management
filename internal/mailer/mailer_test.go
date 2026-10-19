package mailer

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessage_PreventsHeaderInjection(t *testing.T) {
	msg := string(BuildMessage("school@x", "victim@x\r\nBcc: evil@x", "Hi\nBcc: evil@x", "line1\nline2", time.Unix(0, 0)))
	if strings.Contains(msg, "\r\nBcc:") {
		t.Fatalf("header injection not prevented:\n%s", msg)
	}
	if !strings.Contains(msg, "line1\r\nline2") {
		t.Error("body newlines should be normalised to CRLF")
	}
}

func TestBuildMessage_EncodesUnicodeSubject(t *testing.T) {
	msg := string(BuildMessage("a@x", "b@x", "Оценка", "", time.Unix(0, 0)))
	if !strings.Contains(msg, "Subject: =?utf-8?q?") {
		t.Errorf("expected Q-encoded subject, got:\n%s", msg)
	}
}
