package main

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestQuotedMessageID(t *testing.T) {
	quote := &waE2E.ContextInfo{StanzaID: proto.String("3EB0TARGET")}

	cases := map[string]*waE2E.Message{
		"plain text stands alone":   {Conversation: proto.String("hi")},
		"reply in extended text":    {ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("yes"), ContextInfo: quote}},
		"reply on an image caption": {ImageMessage: &waE2E.ImageMessage{Caption: proto.String("this one"), ContextInfo: quote}},
		"reaction names its target": {ReactionMessage: &waE2E.ReactionMessage{Text: proto.String("👌"), Key: &waCommon.MessageKey{ID: proto.String("3EB0TARGET")}}},
		"reply inside an ephemeral envelope": {EphemeralMessage: &waE2E.FutureProofMessage{Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("yes"), ContextInfo: quote},
		}}},
	}
	want := map[string]string{
		"plain text stands alone":            "",
		"reply in extended text":             "3EB0TARGET",
		"reply on an image caption":          "3EB0TARGET",
		"reaction names its target":          "3EB0TARGET",
		"reply inside an ephemeral envelope": "3EB0TARGET",
	}
	for name, msg := range cases {
		if got := quotedMessageID(msg); got != want[name] {
			t.Errorf("%s: got %q want %q", name, got, want[name])
		}
	}
	if got := quotedMessageID(nil); got != "" {
		t.Errorf("nil: got %q", got)
	}
}
