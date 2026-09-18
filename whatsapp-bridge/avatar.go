package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Profile pictures. whatsmeow can ask WhatsApp for a contact's or group's picture
// (GetProfilePictureInfo) but the bridge never exposed it, so every consumer showed
// initials. This endpoint fetches the picture on demand and persists it under the
// media folder, mirroring /api/media/download: the caller gets a local path, not
// bytes, and asks again when it wants a refresh (WhatsApp returns the same picture
// ID while the picture is unchanged, so the caller can skip a re-download).
//
//	GET /api/avatar?jid=<jid>[&full=1]
//	→ 200 {"jid": ..., "path": ..., "mime": "image/jpeg", "size": N, "picture_id": "..."}
//	→ 404 {"error": "profile_picture_not_set" | "profile_picture_unauthorized"}
//
// Read-only from WhatsApp's point of view: fetching a picture is a query the phone
// app makes constantly; it sends nothing and marks nothing.

const avatarFetchTimeout = 30 * time.Second

type avatarResponse struct {
	JID       string `json:"jid"`
	Path      string `json:"path"`
	Mime      string `json:"mime"`
	Size      int64  `json:"size"`
	PictureID string `json:"picture_id"`
	Preview   bool   `json:"preview"`
}

// FetchProfilePicture resolves the picture for jid and stores it as
// <MediaPath>/avatars/<safe jid>[-full].jpg, overwriting any earlier copy so the
// file on disk is always the newest picture WhatsApp served.
func (b *Bridge) FetchProfilePicture(ctx context.Context, jidStr string, preview bool) (*avatarResponse, error) {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	if b.client == nil || b.client.Store.ID == nil {
		return nil, errors.New("not paired")
	}

	info, err := b.client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: preview})
	if err != nil {
		return nil, err
	}
	if info == nil || info.URL == "" {
		return nil, whatsmeow.ErrProfilePictureNotSet
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: status %d", resp.StatusCode)
	}

	dir := filepath.Join(b.cfg.MediaPath, "avatars")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	name := safeMediaStem(jid.String())
	if !preview {
		name += "-full"
	}
	path := filepath.Join(dir, name+".jpg")
	tmp := path + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	size, err := io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		mime = "image/jpeg" // WhatsApp serves profile pictures as JPEG
	}
	return &avatarResponse{JID: jid.String(), Path: path, Mime: mime, Size: size, PictureID: info.ID, Preview: preview}, nil
}

func (s *Server) handleAvatar(w http.ResponseWriter, r *http.Request) {
	jid := r.URL.Query().Get("jid")
	if jid == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "jid required"})
		return
	}
	preview := r.URL.Query().Get("full") != "1"

	ctx, cancel := context.WithTimeout(r.Context(), avatarFetchTimeout)
	defer cancel()

	out, err := s.bridge.FetchProfilePicture(ctx, jid, preview)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, out)
	case errors.Is(err, whatsmeow.ErrProfilePictureNotSet):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "profile_picture_not_set"})
	case errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "profile_picture_unauthorized"})
	case strings.HasPrefix(err.Error(), "invalid jid"):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "avatar_fetch_failed", Details: err.Error()})
	}
}
