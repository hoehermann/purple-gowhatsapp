//go:build gometa

package main

// Received Facebook files: pictures, videos, voice messages, documents and stickers. Each is
// downloaded when it comes, on its own goroutine, and kept under
// <purple user dir>/gometa/attachments/<account>/; the chat then shows a picture inline and any
// other file as a link to it, with the caption after it (gometabridge.c). A file already kept is
// not downloaded again: it is named after its content hash (encrypted chats) or its Facebook id
// (plain chats), and each login's inbox repeats the latest messages.
//
// This does not use the WhatsApp protocol's attachment handling (glue/handle_attachment.c), which
// depends on that protocol's connection data and account options.

/*
#include "gometabridge.h"
*/
import "C"

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/types/events"
)

// gometaDownloads bounds how many files are downloaded at once (a login's inbox can name many).
var gometaDownloads = make(chan struct{}, 3)

// gometaGo runs an attachment's download (a variable so the tests can run it in line).
var gometaGo = func(f func()) { go f() }

// gometaFileName is the relative name a file is kept under: hash and extension, or for a document
// with a name of its own, that name (made safe as a file name) in a folder named after the hash.
func gometaFileName(hash, name, extension string) string {
	if name != "" {
		if safe := sanitize_file_name(name); safe != "" {
			return filepath.Join(hash, safe+extension)
		}
	}
	return hash + extension
}

// attachmentDir is where this account's files are kept.
func (h *gometaHandler) attachmentDir() string {
	return filepath.Join(h.purpleUserDir, "gometa", "attachments", sanitize_file_name(h.username))
}

// receiveAttachment keeps a file of message m, fetching it unless it is kept already, and then shows
// it. When it cannot be had, the caption alone is shown, if there is one.
func (h *gometaHandler) receiveAttachment(m gometaIncoming, name, mimetype string, fetch func(ctx context.Context) ([]byte, error)) {
	gometaGo(func() {
		path := filepath.Join(h.attachmentDir(), name)
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			gometaDownloads <- struct{}{}
			ctx, cancel := context.WithTimeout(h.ctx, 3*time.Minute)
			data, err := fetch(ctx)
			cancel()
			<-gometaDownloads
			if err == nil {
				err = gometaWriteFile(path, data)
			}
			if err != nil {
				h.logger.Warn().Err(err).Str("mime", mimetype).Msg("Facebook attachment not downloaded")
				if m.text != "" {
					h.out().message(m)
				}
				return
			}
		}
		h.out().attachment(m, path, mimetype)
	})
}

// handleE2EEMedia takes an encrypted message's picture, video, voice message, document or sticker.
// It tells whether the content was one.
func (h *gometaHandler) handleE2EEMedia(content *waConsumerApplication.ConsumerApplication_Content, evt *events.FBMessage, chatFbid, senderFbid int64, name string) bool {
	media, ok := gometaE2EEMediaOf(content)
	if !ok {
		return false
	}
	id := string(evt.Info.ID)
	if id != "" {
		h.mu.Lock()
		h.e2eeMsgMeta[id] = e2eeMsgInfo{fromMe: evt.Info.IsFromMe, sender: strconv.FormatInt(senderFbid, 10), text: media.caption}
		h.mu.Unlock()
	}
	h.addContact(chatFbid, name)
	m := gometaIncoming{conv: strconv.FormatInt(chatFbid, 10), who: strconv.FormatInt(senderFbid, 10), name: name,
		text: media.caption, id: id, ts: evt.Info.Timestamp.Unix(), isOutgoing: evt.Info.IsFromMe}
	integral := media.transport.GetIntegral()
	h.receiveAttachment(m, media.fileName(), media.mimetype, func(ctx context.Context) ([]byte, error) {
		return h.net().downloadFB(ctx, integral, media.mediaType)
	})
	h.noteUnreadE2EE(chatFbid, evt.Info) // for the read receipt when the chat is shown
	return true
}

// postPlainMedia takes a plain chat's message with files: each is kept and shown, the message's text
// with the first.
func (h *gometaHandler) postPlainMedia(m gometaIncoming, files []gometaFBAttachment) {
	for i, a := range files {
		a := a
		part := m
		if i > 0 {
			part.text, part.id = "", ""
		}
		h.receiveAttachment(part, a.fileName(), a.mime, func(ctx context.Context) ([]byte, error) {
			return h.net().fetch(ctx, a.url, a.mime)
		})
	}
}

func gometaWriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// gometaNotifyAttachment shows a received file kept at path; m.text is its caption (markup).
func gometaNotifyAttachment(account *PurpleAccount, m gometaIncoming, path, mimetype string) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_attachment), timestamp: C.time_t(m.ts)}
	msg.conv = C.CString(m.conv)
	msg.who = C.CString(m.who)
	if m.name != "" {
		msg.name = C.CString(m.name)
	}
	if m.text != "" {
		msg.text = C.CString(m.text)
	}
	if m.id != "" {
		msg.id = C.CString(m.id)
	}
	msg.path = C.CString(path)
	if mimetype != "" {
		msg.mimetype = C.CString(mimetype)
	}
	if m.isGroup {
		msg.isGroup = 1
	}
	if m.isOutgoing {
		msg.isOutgoing = 1
	}
	C.gometa_process_message(msg)
}
