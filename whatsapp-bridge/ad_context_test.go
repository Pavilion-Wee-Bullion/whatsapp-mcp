package main

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestAdContextFromAClickToWhatsAppMessage(t *testing.T) {
	m := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text: proto.String("Hello! Can I get more info on this?"),
		ContextInfo: &waE2E.ContextInfo{
			ConversionSource:           proto.String("FB_Ads"),
			EntryPointConversionSource: proto.String("ctwa_ad"),
			EntryPointConversionApp:    proto.String("facebook"),
			ExternalAdReply: &waE2E.ContextInfo_ExternalAdReplyInfo{
				Title:                         proto.String("Contact Kami Sekarang"),
				Body:                          proto.String("Perlukan wang tunai"),
				SourceType:                    proto.String("ad"),
				SourceID:                      proto.String("120200000000000"),
				SourceURL:                     proto.String("https://fb.me/abc"),
				CtwaClid:                      proto.String("clid-1"),
				Thumbnail:                     []byte{1, 2, 3},
				AutomatedGreetingMessageShown: proto.Bool(true),
			},
		},
	}}

	got := adContextJSON(m)
	var a adContext
	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatalf("not JSON: %q (%v)", got, err)
	}
	if a.Title != "Contact Kami Sekarang" || a.SourceID != "120200000000000" || a.SourceURL != "https://fb.me/abc" || a.CtwaClid != "clid-1" {
		t.Errorf("ad card fields lost: %+v", a)
	}
	if a.ConversionSource != "FB_Ads" || a.EntryPointConversionApp != "facebook" {
		t.Errorf("conversion fields lost: %+v", a)
	}
	if a.ThumbnailBase64 != "AQID" {
		t.Errorf("thumbnail not kept inline: %q", a.ThumbnailBase64)
	}
	if a.ThumbnailBytes != 3 || a.AutomatedGreetingMessageShown == nil || !*a.AutomatedGreetingMessageShown {
		t.Errorf("thumbnail count or greeting flag lost: %+v", a)
	}
}

func TestAdContextIsReadThroughAnEnvelope(t *testing.T) {
	inner := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{ContextInfo: &waE2E.ContextInfo{
		ExternalAdReply: &waE2E.ContextInfo_ExternalAdReplyInfo{Title: proto.String("Gold bar")},
	}}}
	m := &waE2E.Message{EphemeralMessage: &waE2E.FutureProofMessage{Message: inner}}

	if got := adContextJSON(m); got != `{"title":"Gold bar"}` {
		t.Errorf("got %q", got)
	}
}

func TestNoAdContextOnOrdinaryMessages(t *testing.T) {
	cases := map[string]*waE2E.Message{
		"nil":        nil,
		"plain text": {Conversation: proto.String("hi")},
		"a reply":    {ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("ok"), ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("ABC")}}},
	}
	for name, m := range cases {
		if got := adContextJSON(m); got != "" {
			t.Errorf("%s: got %q, want empty", name, got)
		}
		if nullableAd(adContextJSON(m)) != nil {
			t.Errorf("%s: an empty ad must be stored as NULL", name)
		}
	}
}

func TestReplyQuoteStillReadAfterTheContextInfoSplit(t *testing.T) {
	m := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("ABC")}}}
	if got := quotedMessageID(m); got != "ABC" {
		t.Errorf("quotedMessageID = %q", got)
	}
}

func TestMigration008AddsTheAdColumnToAnExistingStore(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "upgrade.db"))
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	// A store as the office bridges have it today: through 007, with a message already in it.
	applyMigrationsUpTo(t, db, 7)
	if _, err := db.Exec(`INSERT INTO messages (id, chat_jid, timestamp, type, content_text, is_from_me) VALUES ('OLD-1', 'x@s.whatsapp.net', 1787000000, 'text', 'hi', 0)`); err != nil {
		t.Fatalf("seed pre-008 row: %v", err)
	}
	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations to head: %v", err)
	}

	var old sql.NullString
	if err := db.QueryRow(`SELECT ad_context_json FROM messages WHERE id = 'OLD-1'`).Scan(&old); err != nil {
		t.Fatalf("column missing after 008: %v", err)
	}
	if old.Valid {
		t.Errorf("an existing row must stay NULL, got %q", old.String)
	}
	if _, err := db.Exec(`INSERT INTO messages (id, chat_jid, timestamp, type, content_text, is_from_me, ad_context_json) VALUES ('NEW-1', 'x@s.whatsapp.net', 1787000001, 'text', 'hi', 0, ?)`, `{"title":"Gold bar"}`); err != nil {
		t.Fatalf("insert with an ad: %v", err)
	}
}

func TestAnOversizedAdThumbnailIsCountedNotKept(t *testing.T) {
	m := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{ContextInfo: &waE2E.ContextInfo{
		ExternalAdReply: &waE2E.ContextInfo_ExternalAdReplyInfo{Title: proto.String("x"), Thumbnail: make([]byte, maxAdThumbnailBytes+1)},
	}}}
	var a adContext
	if err := json.Unmarshal([]byte(adContextJSON(m)), &a); err != nil {
		t.Fatal(err)
	}
	if a.ThumbnailBase64 != "" || a.ThumbnailBytes != maxAdThumbnailBytes+1 {
		t.Errorf("got base64 len %d, bytes %d", len(a.ThumbnailBase64), a.ThumbnailBytes)
	}
}
