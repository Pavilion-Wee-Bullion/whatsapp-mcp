package main

import (
	"encoding/base64"
	"encoding/json"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// adContext is what a message says about the Meta ad a chat was opened from (click-to-WhatsApp). WhatsApp puts it in
// the message's ContextInfo: the ad card (ExternalAdReply: the ad's title, body, link and source) and the conversion
// fields beside it. Which message carries it (the customer's first message or the automated greeting) and which
// fields are filled is not documented, so every readable field is kept and the binary ones are only counted.
type adContext struct {
	Title                         string `json:"title,omitempty"`
	Body                          string `json:"body,omitempty"`
	MediaType                     string `json:"media_type,omitempty"`
	ThumbnailURL                  string `json:"thumbnail_url,omitempty"`
	MediaURL                      string `json:"media_url,omitempty"`
	OriginalImageURL              string `json:"original_image_url,omitempty"`
	AdPreviewURL                  string `json:"ad_preview_url,omitempty"`
	SourceType                    string `json:"source_type,omitempty"`
	SourceID                      string `json:"source_id,omitempty"`
	SourceURL                     string `json:"source_url,omitempty"`
	SourceApp                     string `json:"source_app,omitempty"`
	CtwaClid                      string `json:"ctwa_clid,omitempty"`
	Ref                           string `json:"ref,omitempty"`
	AdType                        string `json:"ad_type,omitempty"`
	GreetingMessageBody           string `json:"greeting_message_body,omitempty"`
	AutomatedGreetingMessageShown *bool  `json:"automated_greeting_message_shown,omitempty"`
	ShowAdAttribution             *bool  `json:"show_ad_attribution,omitempty"`
	ThumbnailBytes                int    `json:"thumbnail_bytes,omitempty"`
	// The card's image travels inside the message; Meta's image URLs above carry an expiry (oe=), so this copy is
	// the one that still shows the ad later. Skipped past maxAdThumbnailBytes so one odd ad cannot bloat a row.
	ThumbnailBase64 string `json:"thumbnail_base64,omitempty"`

	ConversionSource                   string `json:"conversion_source,omitempty"`
	ConversionDataBytes                int    `json:"conversion_data_bytes,omitempty"`
	EntryPointConversionSource         string `json:"entry_point_conversion_source,omitempty"`
	EntryPointConversionApp            string `json:"entry_point_conversion_app,omitempty"`
	EntryPointConversionExternalSource string `json:"entry_point_conversion_external_source,omitempty"`
	EntryPointConversionExternalMedium string `json:"entry_point_conversion_external_medium,omitempty"`
	CtwaSignals                        string `json:"ctwa_signals,omitempty"`
	CtwaPayloadBytes                   int    `json:"ctwa_payload_bytes,omitempty"`
}

// maxAdThumbnailBytes bounds the inline thumbnail kept with an ad; a real one is about 2 KB.
const maxAdThumbnailBytes = 64 * 1024

// adContextJSON returns the ad a message carries as compact JSON for messages.ad_context_json, or "" when it carries
// none (a plain reply's ContextInfo has a quote but no ad, and stays "").
func adContextJSON(raw *waE2E.Message) string {
	if raw == nil {
		return ""
	}
	ctx := contextInfoOf(unwrapEnvelope(raw))
	if ctx == nil {
		return ""
	}
	ad := ctx.GetExternalAdReply()
	a := adContext{
		ConversionSource:                   ctx.GetConversionSource(),
		ConversionDataBytes:                len(ctx.GetConversionData()),
		EntryPointConversionSource:         ctx.GetEntryPointConversionSource(),
		EntryPointConversionApp:            ctx.GetEntryPointConversionApp(),
		EntryPointConversionExternalSource: ctx.GetEntryPointConversionExternalSource(),
		EntryPointConversionExternalMedium: ctx.GetEntryPointConversionExternalMedium(),
		CtwaSignals:                        ctx.GetCtwaSignals(),
		CtwaPayloadBytes:                   len(ctx.GetCtwaPayload()),
	}
	if ad != nil {
		a.Title = ad.GetTitle()
		a.Body = ad.GetBody()
		if ad.MediaType != nil {
			a.MediaType = ad.GetMediaType().String()
		}
		a.ThumbnailURL = ad.GetThumbnailURL()
		a.MediaURL = ad.GetMediaURL()
		a.OriginalImageURL = ad.GetOriginalImageURL()
		a.AdPreviewURL = ad.GetAdPreviewURL()
		a.SourceType = ad.GetSourceType()
		a.SourceID = ad.GetSourceID()
		a.SourceURL = ad.GetSourceURL()
		a.SourceApp = ad.GetSourceApp()
		a.CtwaClid = ad.GetCtwaClid()
		a.Ref = ad.GetRef()
		if ad.AdType != nil {
			a.AdType = ad.GetAdType().String()
		}
		a.GreetingMessageBody = ad.GetGreetingMessageBody()
		a.AutomatedGreetingMessageShown = ad.AutomatedGreetingMessageShown
		a.ShowAdAttribution = ad.ShowAdAttribution
		a.ThumbnailBytes = len(ad.GetThumbnail())
		if n := len(ad.GetThumbnail()); n > 0 && n <= maxAdThumbnailBytes {
			a.ThumbnailBase64 = base64.StdEncoding.EncodeToString(ad.GetThumbnail())
		}
	}
	if a == (adContext{}) {
		return ""
	}
	out, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(out)
}

// nullableAd stores "" as NULL, so a message without an ad leaves the column empty.
func nullableAd(s string) any {
	if s == "" {
		return nil
	}
	return s
}
