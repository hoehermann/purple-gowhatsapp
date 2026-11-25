//go:build gometa

package main

// Facebook Messenger E2EE (encrypted threads) support. Messenger's end-to-end encryption rides
// WhatsApp's Signal transport ("encrypted over WA"), so messagix exposes a whatsmeow client for it
// via RegisterE2EE + PrepareE2EEClient. Encrypted threads CANNOT use the plaintext SendMessageTask
// (Facebook rejects it with HandleInvalidSendToOpen) — they must go through this whatsmeow client.
//
// The whatsmeow device (Signal identity + prekeys) is persisted in a per-account SQLite DB (pure-Go
// modernc driver, registered by login.go) so the E2EE session survives restarts. This client is
// entirely separate from the WhatsApp account's own whatsmeow client (different device + DB).

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/mautrix-meta/pkg/messagix/methods"
	"go.mau.fi/whatsmeow/proto/waArmadilloApplication"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMsgApplication"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

const gometaWADeviceSetting = "gometa_wa_jid"

// gometaE2EEStorePath is an account's E2EE store, in the purple user directory. The username is
// made a single file name, as for the attachment folder.
func gometaE2EEStorePath(purpleUserDir, username string) string {
	return filepath.Join(purpleUserDir, "gometa-e2ee-"+sanitize_file_name(username)+".db")
}

// gometaE2EEStoreURI is the SQLite URI of a store path. SQLite decodes %HH in a URI's file name
// and ends it at '#' (sanitize_file_name has already replaced '?'), so those are escaped.
func gometaE2EEStoreURI(path string) string {
	return "file:" + strings.NewReplacer("%", "%25", "#", "%23", "?", "%3f").Replace(path)
}

// connectE2EE registers (once) and connects the whatsmeow client that carries Messenger's
// encrypted threads. Best-effort: a failure only disables encrypted threads, not plaintext ones.
func (h *gometaHandler) connectE2EE(fbid int64) error {
	dbLog := waLog.Zerolog(h.logger.With().Str("component", "e2ee-db").Logger())
	addr := gometaE2EEStoreURI(gometaE2EEStorePath(h.purpleUserDir, h.username)) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", addr)
	if err != nil {
		return fmt.Errorf("open e2ee db: %w", err)
	}
	db.SetMaxOpenConns(1)
	container := sqlstore.NewWithDB(db, "sqlite", dbLog)
	if err := container.Upgrade(h.ctx); err != nil {
		return fmt.Errorf("upgrade e2ee db: %w", err)
	}

	// Reuse the saved device if we have one, otherwise register a fresh one.
	var device *store.Device
	if jidStr := gometaGetSetting(h.account, gometaWADeviceSetting); jidStr != "" {
		if jid, perr := waTypes.ParseJID(jidStr); perr == nil {
			device, _ = container.GetDevice(h.ctx, jid)
		}
	}
	isNew := device == nil
	if isNew {
		device = container.NewDevice()
	}
	if suggested := h.client.MessengerLite.GetSuggestedDeviceID(); suggested != uuid.Nil {
		device.FacebookUUID = suggested
	}
	h.client.SetDevice(device)

	if isNew {
		h.logger.Info().Msg("registering new E2EE device")
		if err := h.client.RegisterE2EE(h.ctx, fbid); err != nil {
			return fmt.Errorf("register e2ee: %w", err)
		}
		if device.ID != nil {
			gometaSetSetting(h.account, gometaWADeviceSetting, device.ID.String())
		}
		if err := device.Save(h.ctx); err != nil {
			return fmt.Errorf("save e2ee device: %w", err)
		}
	}

	e2ee, err := h.client.PrepareE2EEClient()
	if err != nil {
		return fmt.Errorf("prepare e2ee client: %w", err)
	}
	e2ee.AddEventHandler(func(evt any) {
		// Online once connected, so typing comes to us (gometa_presence.go).
		if _, ok := evt.(*events.Connected); ok {
			go h.announceAvailable(e2ee)
		}
		h.e2eeEventHandler(evt)
	})
	if err := e2ee.Connect(); err != nil {
		return fmt.Errorf("connect e2ee socket: %w", err)
	}
	h.e2ee = e2ee
	h.logger.Info().Msg("E2EE (whatsmeow) client connected")
	return nil
}

// e2eeEventHandler receives events from the encrypted (whatsmeow) socket.
func (h *gometaHandler) e2eeEventHandler(rawEvt any) {
	switch evt := rawEvt.(type) {
	case *events.FBMessage:
		h.handleE2EEMessage(evt)
	case *events.Connected:
		h.logger.Info().Msg("E2EE socket connected")
	case *events.ChatPresence:
		h.handleChatPresence(evt) // typing (gometa_presence.go)
	case *events.LoggedOut:
		h.logger.Warn().Msg("E2EE socket logged out")
	default:
		// Surface anything we don't handle (e.g. if reactions ever arrive as a distinct event type
		// rather than an FBMessage) so it's visible in the log instead of silently dropped.
		h.logger.Debug().Str("type", fmt.Sprintf("%T", rawEvt)).Msg("unhandled E2EE event")
	}
}

// handleE2EEMessage turns an encrypted message into a buddy + incoming message, and records the
// chat/sender as E2EE so future sends to them route over the encrypted transport.
func (h *gometaHandler) handleE2EEMessage(evt *events.FBMessage) {
	chatFbid, _ := strconv.ParseInt(evt.Info.Chat.User, 10, 64)
	senderFbid, _ := strconv.ParseInt(evt.Info.Sender.User, 10, 64)
	if chatFbid == 0 {
		return
	}
	h.mu.Lock()
	h.e2eeContacts[chatFbid] = true
	if senderFbid != 0 {
		h.e2eeContacts[senderFbid] = true
	}
	name := h.contactName[senderFbid]
	// Drop the echo of a message we just sent from here (tracked by the id we passed to SendFBMessage).
	var mine bool
	if otid, err := strconv.ParseInt(string(evt.Info.ID), 10, 64); err == nil {
		mine = h.sentOtids[otid]
		delete(h.sentOtids, otid)
	}
	h.mu.Unlock()
	if mine {
		return
	}

	var text string
	switch msg := evt.Message.(type) {
	case *waArmadilloApplication.Armadillo:
		// A shared location or link comes as "extended content" (gometa_receive.go).
		text = gometaArmadilloText(msg.GetPayload().GetContent().GetExtendedContentMessage())
	case *waConsumerApplication.ConsumerApplication:
		// "Delete for everyone" comes as application data, an edit as content; both name the
		// message they change by its id (gometa_edits.go).
		if revoke := msg.GetPayload().GetApplicationData().GetRevoke(); revoke != nil {
			h.handleE2EERevoke(evt, revoke.GetKey())
			return
		}
		content := msg.GetPayload().GetContent()
		if edit := content.GetEditMessage(); edit != nil {
			h.handleE2EEEdit(evt, edit.GetKey(), edit.GetMessage().GetText())
			return
		}
		// In an encrypted thread a reaction comes as a ConsumerApplication with a ReactionMessage,
		// not as the LSUpsertReaction table of a plain thread. Key.ID is the message reacted to,
		// Text the emoji ("" when the reaction was taken back).
		if rm := content.GetReactionMessage(); rm != nil {
			if targetID := rm.GetKey().GetID(); targetID != "" {
				h.out().reaction(strconv.FormatInt(chatFbid, 10), targetID,
					strconv.FormatInt(senderFbid, 10), rm.GetText())
			}
			return
		}
		// A picture, video, voice message, document or sticker (gometa_attachment.go).
		if h.handleE2EEMedia(content, evt, chatFbid, senderFbid, name) {
			return
		}
		// A contact card, a text with a link preview, and a location, are texts (gometa_receive.go).
		if contact, ok := gometaE2EEContact(content); ok {
			text = contact
		} else {
			text = gometaE2EEText(content)
		}
	}
	if text == "" {
		return
	}
	// Record who sent this message so a later reaction over E2EE can build a correct MessageKey
	// (FromMe / sender). Keyed by the message id (matches the serviceMessageId we store below).
	if id := string(evt.Info.ID); id != "" {
		h.mu.Lock()
		h.e2eeMsgMeta[id] = e2eeMsgInfo{fromMe: evt.Info.IsFromMe, sender: strconv.FormatInt(senderFbid, 10), text: text}
		h.mu.Unlock()
	}
	// An encrypted reply names the message it answers in the MessageApplication metadata (its
	// StanzaID is that message's id); its text and author come from the per-message cache.
	var quotedText, quotedFrom string
	if qm := evt.FBApplication.GetMetadata().GetQuotedMessage(); qm != nil {
		if quotedID := qm.GetStanzaID(); quotedID != "" {
			h.mu.Lock()
			if meta, ok := h.e2eeMsgMeta[quotedID]; ok {
				quotedText = meta.text
				if sf, perr := strconv.ParseInt(meta.sender, 10, 64); perr == nil {
					quotedFrom = h.contactName[sf]
				}
			}
			h.mu.Unlock()
		}
	}
	h.addContact(chatFbid, name)
	h.notifyMessage(strconv.FormatInt(chatFbid, 10), strconv.FormatInt(senderFbid, 10),
		name, text, evt.Info.ID, evt.Info.Timestamp.Unix(), false, evt.Info.IsFromMe, quotedText, quotedFrom)
	h.noteUnreadE2EE(chatFbid, evt.Info) // for the read receipt when the chat is shown
}

// sendE2EE sends a message over the encrypted (whatsmeow) transport to a Messenger thread.
func (h *gometaHandler) sendE2EE(threadID int64, text string) error {
	to := waTypes.JID{User: strconv.FormatInt(threadID, 10), Server: waTypes.MessengerServer}
	msg := &waConsumerApplication.ConsumerApplication{
		Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{
				Content: &waConsumerApplication.ConsumerApplication_Content{
					Content: &waConsumerApplication.ConsumerApplication_Content_MessageText{
						MessageText: &waCommon.MessageText{Text: proto.String(text)},
					},
				},
			},
		},
	}
	otid := methods.GenerateEpochID()
	otidStr := strconv.FormatInt(otid, 10)
	h.mu.Lock()
	h.sentOtids[otid] = true
	// Our own message, so a later edit or reply over E2EE knows it is ours and can quote it.
	h.e2eeMsgMeta[otidStr] = e2eeMsgInfo{fromMe: true, sender: strconv.FormatInt(h.selfID, 10), text: text}
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()
	err := h.net().sendFB(ctx, to, msg, &waMsgApplication.MessageApplication_Metadata{}, waTypes.MessageID(otidStr))
	if err != nil {
		h.mu.Lock()
		delete(h.sentOtids, otid) // nothing went, so nothing comes back
		delete(h.e2eeMsgMeta, otidStr)
		h.mu.Unlock()
		return err
	}
	// The id we sent with is this message's id, which a reaction to it names.
	h.index.add(gometaIncoming{conv: to.User, who: strconv.FormatInt(h.selfID, 10), text: text, id: otidStr, isOutgoing: true})
	return nil
}
