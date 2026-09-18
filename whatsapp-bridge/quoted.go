package main

import "go.mau.fi/whatsmeow/proto/waE2E"

// quotedMessageID returns the ID of the message this one refers to, or "" when it
// stands alone. Two protocol shapes carry that link: a reply quotes its target in
// ContextInfo.StanzaID (present on every content type that can be a reply), and a
// reaction names its target in ReactionMessage.Key. The messages table has had a
// quoted_message_id column since the first schema but nothing wrote it on receive —
// only the outbound sends path used the column — so every consumer saw "" and could
// neither show a quote nor attach a reaction to the message it was for.
func quotedMessageID(raw *waE2E.Message) string {
	if raw == nil {
		return ""
	}
	m := unwrapEnvelope(raw)

	if r := m.GetReactionMessage(); r != nil {
		return r.GetKey().GetID()
	}

	var ctx *waE2E.ContextInfo
	switch {
	case m.GetExtendedTextMessage() != nil:
		ctx = m.GetExtendedTextMessage().GetContextInfo()
	case m.GetImageMessage() != nil:
		ctx = m.GetImageMessage().GetContextInfo()
	case m.GetVideoMessage() != nil:
		ctx = m.GetVideoMessage().GetContextInfo()
	case m.GetAudioMessage() != nil:
		ctx = m.GetAudioMessage().GetContextInfo()
	case m.GetDocumentMessage() != nil:
		ctx = m.GetDocumentMessage().GetContextInfo()
	case m.GetStickerMessage() != nil:
		ctx = m.GetStickerMessage().GetContextInfo()
	case m.GetLocationMessage() != nil:
		ctx = m.GetLocationMessage().GetContextInfo()
	case m.GetLiveLocationMessage() != nil:
		ctx = m.GetLiveLocationMessage().GetContextInfo()
	case m.GetContactMessage() != nil:
		ctx = m.GetContactMessage().GetContextInfo()
	case m.GetContactsArrayMessage() != nil:
		ctx = m.GetContactsArrayMessage().GetContextInfo()
	case m.GetPollCreationMessage() != nil:
		ctx = m.GetPollCreationMessage().GetContextInfo()
	case m.GetGroupInviteMessage() != nil:
		ctx = m.GetGroupInviteMessage().GetContextInfo()
	case m.GetButtonsMessage() != nil:
		ctx = m.GetButtonsMessage().GetContextInfo()
	case m.GetListMessage() != nil:
		ctx = m.GetListMessage().GetContextInfo()
	case m.GetTemplateMessage() != nil:
		ctx = m.GetTemplateMessage().GetContextInfo()
	}
	return ctx.GetStanzaID()
}
