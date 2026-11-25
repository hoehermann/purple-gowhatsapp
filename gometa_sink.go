//go:build gometa

package main

// Where the Facebook protocol hands what it received to libpurple (gometaSink), and what it asks of
// the two Facebook connections besides sending a message (gometaWire). The handler goes through
// these instead of calling the C bridge and the clients directly, so a test can stand in for both
// and see what a table or an encrypted event leads to.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/mautrix-meta/pkg/messagix/useragent"
	"go.mau.fi/whatsmeow"
	armadillo "go.mau.fi/whatsmeow/proto"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	"go.mau.fi/whatsmeow/proto/waMsgApplication"
	waTypes "go.mau.fi/whatsmeow/types"
)

// gometaIncoming is one message to show: conv is the thread key, who the sender's fbid.
type gometaIncoming struct {
	conv, who, name, text, id string
	ts                        int64
	isGroup, isOutgoing       bool
	quotedText, quotedFrom    string // the message this one replies to, if any
}

type gometaSink interface {
	buddy(fbid int64, name string) // add or update a contact
	message(m gometaIncoming)
	attachment(m gometaIncoming, path, mimetype string) // a received file, kept at path; m.text is its caption
	edit(id, text string)                               // the message shown under id now reads text
	remove(id string)                                   // the message shown under id was deleted for everyone
	reaction(conv, target, actor, emoji string)         // actor reacted to target ("" emoji: took it back)
	typing(who string, on bool)
	failed(text string, fatal bool)                          // an error the user is shown
	notice(conv string, isGroup bool, text string, ts int64) // a line in a chat that nobody wrote
	members(conv string, fbids []int64)                      // who is in a group chat besides us
	chat(threadKey int64, name string)                       // a group chat, made or named again
}

// gometaPurpleSink is the real one: the gometa C bridge (gometabridge.c). Facebook text is plain,
// so it is escaped for libpurple as the WhatsApp protocol's is (escape_text). Edits, deletions and
// reactions are shown as text in the chat, as the WhatsApp protocol shows them (handle_message.go);
// the recent messages index (gometa_index.go) finds which chat they belong to.
type gometaPurpleSink struct{ h *gometaHandler }

func (s gometaPurpleSink) buddy(fbid int64, name string) { gometaNotifyBuddy(s.h.account, fbid, name) }
func (s gometaPurpleSink) message(m gometaIncoming) {
	s.h.index.add(m)
	if m.quotedText != "" {
		m.text = gometaQuote(m.quotedText) + m.text
	}
	m.text = escape_text(m.text)
	gometaNotifyMessage(s.h.account, m)
}
func (s gometaPurpleSink) attachment(m gometaIncoming, path, mimetype string) {
	s.h.index.add(m)
	m.text = escape_text(m.text)
	gometaNotifyAttachment(s.h.account, m, path, mimetype)
}
func (s gometaPurpleSink) edit(id, text string) {
	if m, ok := s.h.index.editLine(id, text, time.Now().Unix()); ok {
		m.text = escape_text(m.text)
		gometaNotifyMessage(s.h.account, m)
	} else {
		s.h.logger.Debug().Str("id", id).Msg("edit of a message not shown here, dropped")
	}
}
func (s gometaPurpleSink) remove(id string) {
	if m, ok := s.h.index.deleteLine(id, time.Now().Unix()); ok {
		m.text = escape_text(m.text)
		gometaNotifyMessage(s.h.account, m)
	} else {
		s.h.logger.Debug().Str("id", id).Msg("deletion of a message not shown here, dropped")
	}
}
func (s gometaPurpleSink) reaction(conv, target, actor, emoji string) {
	s.h.mu.Lock()
	isGroup := s.h.threadGroup[gometaParseID(conv)]
	name := s.h.contactName[gometaParseID(actor)]
	s.h.mu.Unlock()
	m := s.h.index.reactionLine(conv, target, actor, emoji, time.Now().Unix())
	m.name, m.isGroup, m.isOutgoing = name, isGroup, gometaParseID(actor) == s.h.selfID
	m.text = escape_text(m.text)
	gometaNotifyMessage(s.h.account, m)
}
func (s gometaPurpleSink) failed(text string, fatal bool) {
	gometaReportError(s.h.account, text, fatal)
}
func (s gometaPurpleSink) chat(threadKey int64, name string) {
	gometaNotifyChat(s.h.account, threadKey, name)
}
func (s gometaPurpleSink) members(conv string, fbids []int64) {
	gometaNotifyMembers(s.h.account, conv, fbids)
}
func (s gometaPurpleSink) notice(conv string, isGroup bool, text string, ts int64) {
	gometaNotifySystem(s.h.account, conv, isGroup, escape_text(text), ts)
}
func (s gometaPurpleSink) typing(who string, on bool) { gometaNotifyTyping(s.h.account, who, on) }

func (h *gometaHandler) out() gometaSink {
	if h.sink != nil {
		return h.sink
	}
	return gometaPurpleSink{h}
}

type gometaWire interface {
	// encrypted chats (the whatsmeow client)
	chatPresence(ctx context.Context, to waTypes.JID, composing bool) error
	markRead(ctx context.Context, ids []waTypes.MessageID, ts time.Time, chat, sender waTypes.JID) error
	downloadFB(ctx context.Context, integral *waMediaTransport.WAMediaTransport_Integral, mediaType whatsmeow.MediaType) ([]byte, error)
	sendFB(ctx context.Context, to waTypes.JID, msg armadillo.RealMessageApplicationSub, meta *waMsgApplication.MessageApplication_Metadata, id waTypes.MessageID) error
	// plain chats (the messagix client)
	fbTask(ctx context.Context, task socket.Task, stateless bool) error
	sendTask(ctx context.Context, task socket.Task) (*table.LSTable, error) // the tables the reply carries
	waitSend(ctx context.Context) error                                     // until messages can be sent
	fetch(ctx context.Context, url, mime string) ([]byte, error)
}

func (h *gometaHandler) net() gometaWire {
	if h.wire != nil {
		return h.wire
	}
	return gometaLiveWire{h}
}

type gometaLiveWire struct{ h *gometaHandler }

var errGometaE2EEDown = fmt.Errorf("encrypted chats are not connected")

func (w gometaLiveWire) chatPresence(ctx context.Context, to waTypes.JID, composing bool) error {
	if w.h.e2ee == nil {
		return errGometaE2EEDown
	}
	state := waTypes.ChatPresencePaused
	if composing {
		state = waTypes.ChatPresenceComposing
	}
	return w.h.e2ee.SendChatPresence(ctx, to, state, waTypes.ChatPresenceMediaText)
}

func (w gometaLiveWire) markRead(ctx context.Context, ids []waTypes.MessageID, ts time.Time, chat, sender waTypes.JID) error {
	if w.h.e2ee == nil {
		return errGometaE2EEDown
	}
	return w.h.e2ee.MarkRead(ctx, ids, ts, chat, sender)
}

func (w gometaLiveWire) downloadFB(ctx context.Context, integral *waMediaTransport.WAMediaTransport_Integral, mediaType whatsmeow.MediaType) ([]byte, error) {
	if w.h.e2ee == nil {
		return nil, errGometaE2EEDown
	}
	return w.h.e2ee.DownloadFB(ctx, integral, mediaType)
}

func (w gometaLiveWire) sendFB(ctx context.Context, to waTypes.JID, msg armadillo.RealMessageApplicationSub, meta *waMsgApplication.MessageApplication_Metadata, id waTypes.MessageID) error {
	if w.h.e2ee == nil {
		return errGometaE2EEDown
	}
	_, err := w.h.e2ee.SendFBMessage(ctx, to, msg, meta, whatsmeow.SendRequestExtra{ID: id})
	return err
}

func (w gometaLiveWire) sendTask(ctx context.Context, task socket.Task) (*table.LSTable, error) {
	if w.h.client == nil {
		return nil, fmt.Errorf("not connected to Facebook")
	}
	return w.h.client.ExecuteTasks(ctx, task)
}

func (w gometaLiveWire) waitSend(ctx context.Context) error {
	if w.h.client == nil {
		return fmt.Errorf("not connected to Facebook")
	}
	return w.h.client.WaitUntilCanSendMessages(ctx, 20*time.Second)
}

func (w gometaLiveWire) fbTask(ctx context.Context, task socket.Task, stateless bool) error {
	if w.h.client == nil {
		return fmt.Errorf("not connected to Facebook")
	}
	if stateless {
		return w.h.client.ExecuteStatelessTask(ctx, task)
	}
	_, err := w.h.client.ExecuteTasks(ctx, task)
	return err
}

// gometaMaxDownload bounds a plain chat's attachment, which is read into memory. Facebook's own
// limit for a file sent in Messenger is stated as 100 MB (not verified here).
const gometaMaxDownload = 100 << 20

var gometaHTTP = &http.Client{Timeout: 3 * time.Minute}

// fetch downloads a plain chat's attachment from Facebook's CDN. The URL is signed, so it needs
// no cookies; the headers are those mautrix-meta's mediadl sends.
func (gometaLiveWire) fetch(ctx context.Context, url, mime string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	if strings.HasPrefix(mime, "image/") {
		req.Header.Set("Accept", "image/avif,image/webp,*/*")
	}
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("User-Agent", useragent.UserAgent)
	resp, err := gometaHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, gometaMaxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > gometaMaxDownload {
		return nil, fmt.Errorf("larger than %d bytes", gometaMaxDownload)
	}
	return data, nil
}
