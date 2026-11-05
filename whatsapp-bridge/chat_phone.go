package main

import (
	"regexp"
	"strings"
)

// chatPhoneColumns are the two places a direct chat's number can be read when its JID is the @lid form, which
// carries no number: the phone-JID alias WhatsApp sent beside the LID (jid_aliases), and the contact row's phone.
// They are selected next to each chat in /api/chats and settled by chatPhone.
//
// Both are keyed by the sender as WhatsApp sent it, which keeps the device part when the customer writes from a
// linked device ("X:3@lid", while the chat is "X@lid"), so the device forms of the chat's JID are matched too.
// The range on ':' to ';' (the next character) reads every "X:<device>..." key off the primary key's index.
const chatPhoneColumns = `
	COALESCE((SELECT jid_b FROM jid_aliases
	          WHERE (jid_a = chats.jid OR (jid_a > ` + chatUserSQL + ` || ':' AND jid_a < ` + chatUserSQL + ` || ';' AND jid_a LIKE '%@lid'))
	            AND jid_b LIKE '%@s.whatsapp.net'
	          ORDER BY discovered_at DESC LIMIT 1), ''),
	COALESCE((SELECT phone FROM contacts
	          WHERE (contacts.jid = chats.jid OR (contacts.jid > ` + chatUserSQL + ` || ':' AND contacts.jid < ` + chatUserSQL + ` || ';' AND contacts.jid LIKE '%@lid'))
	            AND COALESCE(phone, '') <> ''
	          ORDER BY updated_at DESC LIMIT 1), '')`

// chatUserSQL is the part of chats.jid before the '@'.
const chatUserSQL = `substr(chats.jid, 1, instr(chats.jid, '@') - 1)`

var phoneDigits = regexp.MustCompile(`^\d{7,15}$`)

// chatPhone is the other party's number for a direct chat, digits only, or "" when the bridge does not know it.
// A phone JID is its own number. An @lid chat takes the alias WhatsApp paired with it, else the contact's phone;
// a contact phone equal to the LID's own digits is the old bug that stored the LID as a phone (see onMessage)
// and is not a number. Groups have no counterparty.
func chatPhone(jid, chatType, aliasJID, contactPhone string) string {
	if chatType == "group" || strings.HasSuffix(jid, "@g.us") {
		return ""
	}
	if user, ok := phoneUser(jid); ok {
		return user
	}
	if !strings.HasSuffix(jid, "@lid") {
		return ""
	}
	if user, ok := phoneUser(aliasJID); ok {
		return user
	}
	lidUser := strings.TrimSuffix(jid, "@lid")
	if i := strings.IndexAny(lidUser, ":."); i >= 0 {
		lidUser = lidUser[:i]
	}
	if p := strings.TrimPrefix(contactPhone, "+"); phoneDigits.MatchString(p) && p != lidUser {
		return p
	}
	return ""
}

// phoneUser reads the number out of a phone-form JID ("60123456789@s.whatsapp.net", with or without a device
// part such as ":12").
func phoneUser(jid string) (string, bool) {
	user, ok := strings.CutSuffix(jid, "@s.whatsapp.net")
	if !ok {
		return "", false
	}
	if i := strings.IndexAny(user, ":."); i >= 0 {
		user = user[:i]
	}
	return user, phoneDigits.MatchString(user)
}
