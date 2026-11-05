-- 008: keep the Meta ad a chat was opened from (click-to-WhatsApp).
--
-- A chat started from a Facebook or Instagram ad carries the ad card in the
-- message's ContextInfo (ExternalAdReply: title, body, link, source, ctwa_clid).
-- The decoder only ever read the text, so every chat's ad was lost. The column
-- holds what ad_context.go extracts, as JSON, NULL when a message carries no ad.
-- Additive: no existing row changes, and older rows stay NULL because the raw
-- message they came from was never kept.

BEGIN;

ALTER TABLE messages ADD COLUMN ad_context_json TEXT;

INSERT OR IGNORE INTO schema_version (version, applied_at, description)
VALUES (8, strftime('%s', 'now'), 'messages.ad_context_json: the Meta ad a chat was opened from');

COMMIT;
