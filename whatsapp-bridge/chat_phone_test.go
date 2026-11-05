package main

import (
	"context"
	"testing"
)

func TestChatPhoneRules(t *testing.T) {
	cases := []struct {
		name, jid, chatType, alias, contactPhone, want string
	}{
		{"phone JID is its own number", "60123456789@s.whatsapp.net", "direct", "", "", "60123456789"},
		{"LID takes the paired phone JID", "85053405147300@lid", "direct", "60186649901@s.whatsapp.net", "", "60186649901"},
		{"device part is dropped", "85053405147300@lid", "direct", "60186649901:12@s.whatsapp.net", "", "60186649901"},
		{"LID falls back to the contact phone", "85053405147300@lid", "direct", "", "60186649901", "60186649901"},
		{"a contact phone that is the LID itself is not a number", "85053405147300@lid", "direct", "", "85053405147300", ""},
		{"LID with nothing known", "85053405147300@lid", "direct", "", "", ""},
		{"groups have no counterparty", "120363000000000000@g.us", "group", "", "60186649901", ""},
		{"junk contact phone", "85053405147300@lid", "direct", "", "not-a-number", ""},
	}
	for _, c := range cases {
		if got := chatPhone(c.jid, c.chatType, c.alias, c.contactPhone); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestListChatsCarriesTheNumberBehindAnLIDChat(t *testing.T) {
	db := undecodedTestDB(t)
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	mustExec(`INSERT INTO chats (jid, chat_type, name, last_message_time, unread_count, created_at, updated_at) VALUES
		('85053405147300@lid', 'direct', 'Aaisyah', 300, 0, 1, 1),
		('62959992746093@lid', 'direct', 'Ngsang', 200, 0, 1, 1),
		('60111111111@s.whatsapp.net', 'direct', 'Phone', 100, 0, 1, 1),
		('11111111111111@lid', 'direct', 'Unknown', 50, 0, 1, 1)`)
	mustExec(`INSERT INTO jid_aliases (jid_a, jid_b, discovered_at, source) VALUES
		('85053405147300@lid', '60186649901@s.whatsapp.net', 10, 'message_sender'),
		('60186649901@s.whatsapp.net', '85053405147300@lid', 10, 'message_sender')`)
	mustExec(`INSERT INTO contacts (jid, phone, push_name, is_business, created_at, updated_at) VALUES
		('62959992746093@lid', '60197639012', 'Ngsang', 0, 1, 1)`)

	chats, err := listChats(context.Background(), db, 20, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range chats {
		got[c.JID] = c.Phone
	}
	want := map[string]string{
		"85053405147300@lid":         "60186649901",
		"62959992746093@lid":         "60197639012",
		"60111111111@s.whatsapp.net": "60111111111",
		"11111111111111@lid":         "",
	}
	for jid, phone := range want {
		if got[jid] != phone {
			t.Errorf("%s: phone %q, want %q", jid, got[jid], phone)
		}
	}
	if len(chats) != 4 || chats[0].JID != "85053405147300@lid" {
		t.Errorf("order or count changed: %+v", chats)
	}
}
