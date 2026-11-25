//go:build gometa

// gometa.go — the Facebook Messenger protocol (prpl-gometa), built with -tags gometa. It shares
// this plug-in's Go runtime with the WhatsApp protocol (same package main): a process can host
// only one Go c-archive, so a separate Go-based Facebook plug-in could not be loaded next to this
// one. All package-level identifiers are gometa-prefixed to stay clear of the WhatsApp code.
//
// Plain threads go through messagix (go.mau.fi/mautrix-meta/pkg/messagix), end-to-end encrypted
// ones through whatsmeow's Facebook client (gometa_e2ee.go).
//
// Login is e-mail + password + interactive 2FA (messagix MessengerLite/Bloks DoLoginSteps): the
// account's username is the e-mail, its password the Facebook password, and a 2FA code is asked
// for with purple_request_input. On success the session cookies are cached in the account
// (setting "gometa_session") so later logins skip the password and 2FA.
package main

/*
#include "gometabridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-meta/pkg/messagix"
	"go.mau.fi/mautrix-meta/pkg/messagix/cookies"
	"go.mau.fi/mautrix-meta/pkg/messagix/methods"
	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/mautrix-meta/pkg/messagix/types"
	"go.mau.fi/whatsmeow"
	"maunium.net/go/mautrix/bridgev2"
)

const gometaSessionSetting = "gometa_session"

// ---- exported prpl callbacks (wired from the C glue's second prpl) ----

//export gometa_go_login
func gometa_go_login(account *PurpleAccount, purpleUserDir *C.char, username *C.char, password *C.char, proxy *C.char) {
	gometaLogin(account, C.GoString(purpleUserDir), C.GoString(username), C.GoString(password), C.GoString(proxy))
}

//export gometa_go_close
func gometa_go_close(account *PurpleAccount) {
	if h, ok := gometaHandlers[account]; ok {
		h.close()
	}
}

// Fired from the libpurple "account-removed" signal (glue/gometa_init.c) when a Facebook account is
// deleted. Closes any live client and deletes only this account's E2EE store.
//
//export gometa_go_account_removed
func gometa_go_account_removed(account *PurpleAccount, purpleUserDir *C.char, username *C.char) {
	gometaAccountRemoved(account, C.GoString(purpleUserDir), C.GoString(username))
}

// gometaAccountRemoved tears down a deleted Facebook-E2EE account: it closes any live messagix + E2EE
// (whatsmeow) client for THIS account, then deletes only this account's per-account E2EE store
// (gometa-e2ee-<username>.db and its -wal/-shm sidecars). The DB path embeds the account username
// (the FB email, exactly as connectE2EE builds it), so removal is strictly scoped to this account and
// never touches another account's store. Unlike WhatsApp's shared whatsmeow.db, this per-account file
// is safe to delete whole. (The messagix E2EE whatsmeow client has MessengerConfig set, so it cannot
// Logout() server-side — deleting the local Signal store is the correct teardown here.)
func gometaAccountRemoved(account *PurpleAccount, purpleUserDir, username string) {
	if h, ok := gometaHandlers[account]; ok {
		h.close() // disconnect e2ee + messagix client, cancel ctx, drop from registry
	}
	if username == "" {
		return
	}
	dbPath := gometaE2EEStorePath(purpleUserDir, username)
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			// Best-effort: log to stderr only. Do NOT route through gometaReportError/gometa_process_message
			// here — those defer a dispatch (purple_timeout_add) that reads the account pointer, but the
			// account is destroyed synchronously right after this "account-removed" signal returns.
			fmt.Fprintf(os.Stderr, "gometa: could not remove E2EE store %s: %v\n", p, err)
		}
	}
}

//export gometa_go_send_message
func gometa_go_send_message(account *PurpleAccount, who *C.char, message *C.char) C.int {
	// Without a request UI (purple_request_get_ui_ops() == NULL) the 2FA prompt is shown in a
	// "Facebook" chat (gometabridge.c), and the reply there is the code, not a message.
	gometaInputMu.Lock()
	ch := gometaInputWaiters[account]
	gometaInputMu.Unlock()
	if ch != nil {
		text, _ := decompose_html(C.GoString(message))
		select {
		case ch <- strings.TrimSpace(text):
		default:
		}
		return 0
	}
	h, ok := gometaHandlers[account]
	if !ok {
		return -107 // ENOTCONN, see libpurple/prpl.h
	}
	text, _ := decompose_html(C.GoString(message))
	go h.sendMessage(C.GoString(who), text)
	return 1
}

// The UI's answer to an interactive login prompt (2FA code). Empty = cancelled.
//
//export gometa_go_submit_input
func gometa_go_submit_input(account *PurpleAccount, value *C.char) {
	gometaInputMu.Lock()
	ch := gometaInputWaiters[account]
	gometaInputMu.Unlock()
	if ch != nil {
		select {
		case ch <- C.GoString(value):
		default:
		}
	}
}

// ---- interactive-input plumbing (2FA/captcha) ----

var (
	gometaInputMu      sync.Mutex
	gometaInputWaiters = make(map[*PurpleAccount]chan string)
)

// promptInput asks the UI for a value and blocks (on a goroutine) until submitted.
func (h *gometaHandler) promptInput(prompt string) string {
	ch := make(chan string, 1)
	gometaInputMu.Lock()
	gometaInputWaiters[h.account] = ch
	gometaInputMu.Unlock()
	defer func() {
		gometaInputMu.Lock()
		delete(gometaInputWaiters, h.account)
		gometaInputMu.Unlock()
	}()
	cPrompt := C.CString(prompt)
	C.gometa_request_input(h.account, cPrompt) // asks on the main thread (gometabridge.c)
	C.free(unsafe.Pointer(cPrompt))
	select {
	case v := <-ch:
		return v
	case <-h.ctx.Done():
		return ""
	}
}

// ---- account settings helpers ----

func gometaGetSetting(account *PurpleAccount, key string) string {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cVal := C.gometa_get_setting(account, cKey)
	if cVal == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cVal))
	return C.GoString(cVal)
}

func gometaSetSetting(account *PurpleAccount, key, value string) {
	cKey := C.CString(key)
	cVal := C.CString(value)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cVal))
	C.gometa_set_setting(account, cKey, cVal)
}

// ---- handler ----

var gometaHandlers = make(map[*PurpleAccount]*gometaHandler)

type gometaHandler struct {
	account  *PurpleAccount
	username string
	client   *messagix.Client
	logger   zerolog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	selfID   int64

	purpleUserDir string

	mu          sync.Mutex
	threadGroup map[int64]bool   // thread key -> is a group chat
	contactName map[int64]string // fbid -> display name
	sentOtids   map[int64]bool   // otids we sent ourselves, to drop their server echo

	// E2EE (whatsmeow) — Messenger encrypted threads ride WhatsApp's Signal transport.
	e2ee         *whatsmeow.Client
	e2eeContacts map[int64]bool // fbids known to use E2EE (learned from encrypted messages)
	// Per encrypted-message metadata (message id -> who sent it). Needed to build the MessageKey when
	// reacting over E2EE (whatsmeow's ReactionMessage requires the original message's FromMe/sender,
	// which isn't otherwise recorded per-message). Populated on both send and receive of E2EE messages.
	e2eeMsgMeta map[string]e2eeMsgInfo

	// What the handler hands to libpurple and asks of the two clients (gometa_sink.go); nil means
	// the real ones. Tests set them.
	sink gometaSink
	wire gometaWire
	// Typing and read receipts (gometa_presence.go), under mu.
	parity gometaParity
	// Who is in each group chat, by thread key (gometa_groups.go), under mu.
	members map[int64]map[int64]bool
	// The messages shown recently, for edits, deletions and reactions (gometa_index.go).
	index gometaIndex
}

type e2eeMsgInfo struct {
	fromMe bool
	sender string // fbid of the original message's sender
	text   string // the message body, so a reply quoting this id can show its text
}

func (h *gometaHandler) eventHandler(ctx context.Context, rawEvt any) {
	switch evt := rawEvt.(type) {
	case *messagix.ConnectedEvent:
		h.logger.Info().Msg("connected (CONNACK)")
		h.notify(C.gometa_message_type_connected, "", "", 0)
	case *messagix.ReconnectedEvent:
		h.notify(C.gometa_message_type_connected, "", "", 0)
	case *messagix.TransientDisconnectEvent:
		h.logger.Warn().Err(evt.Err).Int("attempts", evt.ConnectionAttempts).Msg("socket error")
	case *messagix.PermanentErrorEvent:
		h.notifyError(fmt.Sprintf("Facebook connection failed: %v", evt.Err), true)
	case *table.LSTable:
		h.parseTable(evt)
	default:
		h.logger.Trace().Str("type", fmt.Sprintf("%T", rawEvt)).Msg("unhandled event")
	}
}

// sendMessage sends a message to a thread. `who` is the thread key: for a 1:1 chat it's the
// recipient's fbid (send_im buddy id); for a group it's the thread key resolved from the chat
// conversation name (see gometa_chat_send). Plain threads go as a messagix SendMessageTask,
// encrypted ones over the whatsmeow (E2EE) client.
func (h *gometaHandler) sendMessage(who, text string) {
	threadID, err := strconv.ParseInt(who, 10, 64)
	if err != nil {
		h.notifyError(fmt.Sprintf("Cannot send: invalid recipient %q", who), false)
		return
	}
	// Encrypted threads must go over the whatsmeow (E2EE) transport — a plaintext task to an
	// E2EE thread is silently rejected by Facebook (HandleInvalidSendToOpen).
	h.mu.Lock()
	isE2EE := h.e2eeContacts[threadID]
	h.mu.Unlock()
	if isE2EE {
		if err := h.sendE2EE(threadID, text); err != nil {
			h.logger.Warn().Err(err).Int64("thread", threadID).Msg("e2ee send failed")
			if errors.Is(err, errGometaE2EEDown) {
				h.notifyError("Cannot send: this is an encrypted thread and E2EE isn't connected yet", false)
			} else {
				h.notifyError(fmt.Sprintf("Failed to send encrypted message: %v", err), false)
			}
			return
		}
		h.logger.Info().Int64("thread", threadID).Msg("e2ee message sent")
		return
	}
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()
	if werr := h.net().waitSend(ctx); werr != nil {
		h.notifyError(fmt.Sprintf("Cannot send yet (connecting): %v", werr), false)
		return
	}
	otid := methods.GenerateEpochID()
	h.mu.Lock()
	h.sentOtids[otid] = true
	h.mu.Unlock()
	task := &socket.SendMessageTask{
		ThreadId:         threadID,
		Otid:             otid,
		Source:           table.MESSENGER_INBOX_IN_THREAD,
		InitiatingSource: table.FACEBOOK_INBOX,
		SendType:         table.TEXT,
		SyncGroup:        1,
		Text:             text,
	}
	resp, err := h.net().sendTask(ctx, task)
	if err != nil {
		h.logger.Warn().Err(err).Int64("thread", threadID).Msg("send failed")
		h.notifyError(fmt.Sprintf("Failed to send Facebook message: %v", err), false)
		return
	}
	// A successful send echoes our OTID back in LSReplaceOptimsiticMessage. If it's absent, Facebook
	// rejected the plaintext send — almost always because the thread is actually E2EE (the
	// HandleInvalidSendToOpen case). Auto-retry over the encrypted transport and remember it.
	otidStr := strconv.FormatInt(otid, 10)
	confirmed := false
	if resp != nil {
		for _, r := range resp.LSReplaceOptimsiticMessage {
			if r.OfflineThreadingId == otidStr {
				confirmed = true
				// Our message as sent, so a reaction to it can quote it.
				h.index.add(gometaIncoming{conv: who, who: strconv.FormatInt(h.selfID, 10), text: text,
					id: r.MessageId, isGroup: h.isGroup(threadID), isOutgoing: true})
				break
			}
		}
	}
	if confirmed {
		h.logger.Info().Int64("thread", threadID).Msg("message sent")
		return
	}
	h.mu.Lock()
	delete(h.sentOtids, otid) // no echo will come for a rejected send
	h.e2eeContacts[threadID] = true
	h.mu.Unlock()
	h.logger.Warn().Int64("thread", threadID).Msg("plaintext send not confirmed; treating thread as E2EE")
	if eerr := h.sendE2EE(threadID, text); eerr != nil {
		h.logger.Warn().Err(eerr).Int64("thread", threadID).Msg("e2ee retry failed")
		if errors.Is(eerr, errGometaE2EEDown) {
			h.notifyError("Couldn't send — this looks like an encrypted thread and E2EE isn't connected yet.", false)
		} else {
			h.notifyError(fmt.Sprintf("Failed to send encrypted message: %v", eerr), false)
		}
		return
	}
	h.logger.Info().Int64("thread", threadID).Msg("e2ee message sent (after plaintext rejection)")
}

// isGroup tells whether a thread is a group chat.
func (h *gometaHandler) isGroup(threadKey int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.threadGroup[threadKey]
}

func (h *gometaHandler) close() {
	if h.cancel != nil {
		h.cancel()
	}
	if h.e2ee != nil {
		h.e2ee.Disconnect()
	}
	if h.client != nil {
		h.client.Disconnect()
	}
	delete(gometaHandlers, h.account)
}

func (h *gometaHandler) notify(msgtype C.int, who, text string, ts int64) {
	msg := C.gometa_message_t{account: h.account, msgtype: C.char(msgtype), timestamp: C.time_t(ts)}
	if who != "" {
		msg.who = C.CString(who)
	}
	if text != "" {
		msg.text = C.CString(text)
	}
	C.gometa_process_message(msg)
}

func (h *gometaHandler) notifyError(text string, fatal bool) {
	h.out().failed(text, fatal)
}

func gometaReportError(account *PurpleAccount, text string, fatal bool) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_error), text: C.CString(text)}
	if fatal {
		msg.fatal = 1
	}
	C.gometa_process_message(msg)
}

// ---- LSTable -> purple (contacts, group chats, incoming messages) ----

// parseTable translates a Facebook LSTable (from the initial load or a realtime publish) into
// buddies, group chats and messages. Called from the login goroutine and event goroutines, so
// the shared maps are mutex-guarded.
func (h *gometaHandler) parseTable(tbl *table.LSTable) {
	if tbl == nil {
		return
	}
	// Full contact rows.
	for _, c := range tbl.LSDeleteThenInsertContact {
		h.addContact(c.Id, c.Name)
	}
	// Referenced contacts (arrive when a thread/message names someone); skip our own row.
	for _, c := range tbl.LSVerifyContactRowExists {
		if c.IsSelf {
			continue
		}
		h.addContact(c.ContactId, c.Name)
	}
	// Threads: a 1:1 thread's ThreadKey IS the other user's fbid (ThreadName = their name),
	// so it becomes a buddy; a group thread becomes a chat.
	for _, t := range tbl.LSDeleteThenInsertThread {
		if t.ThreadKey == 0 {
			continue
		}
		isGroup := !t.ThreadType.IsOneToOne()
		h.mu.Lock()
		h.threadGroup[t.ThreadKey] = isGroup
		if h.parity.threadType != nil {
			h.parity.threadType[t.ThreadKey] = t.ThreadType
		}
		// Threads already typed as "encrypted over WA" are E2EE — flag them so sends route
		// over the whatsmeow transport without waiting to receive an encrypted message first.
		if t.ThreadType.IsWhatsApp() {
			h.e2eeContacts[t.ThreadKey] = true
		}
		h.mu.Unlock()
		if isGroup {
			h.notifyChat(t.ThreadKey, t.ThreadName)
		} else {
			h.addContact(t.ThreadKey, t.ThreadName)
		}
	}
	// Group participants become buddies too.
	for _, p := range tbl.LSAddParticipantIdToGroupThread {
		h.addContact(p.ContactId, p.Nickname)
	}
	// Who is in each group, and a group's new name (gometa_groups.go).
	h.handleParticipantTables(tbl)
	h.handleThreadNames(tbl)
	// Which Facebook thread an encrypted chat is, for its read state (gometa_presence.go).
	h.noteHybridThreads(tbl)
	// A 1:1 chat's partner typing (gometa_presence.go).
	h.handleTypingTable(tbl.LSUpdateTypingIndicator)
	// Contact presence (online/offline). Status 0 = offline; non-zero = active/available.
	for _, p := range tbl.LSDeleteThenInsertContactPresence {
		if p.ContactId == 0 {
			continue
		}
		h.notifyPresence(p.ContactId, p.Status != 0)
	}
	// A message's pictures, files, stickers and shared locations or links come in their own tables,
	// by message id; a message with no text of its own is one of those.
	atts := gometaPlainAttachments(tbl)
	for _, m := range tbl.LSInsertMessage {
		h.handleMessage(m, atts[m.MessageId])
	}
	for _, m := range tbl.LSUpsertMessage {
		h.handleMessage(m.ToInsert(), atts[m.MessageId])
	}
	// Edits and deletes, after the messages they may replace (gometa_edits.go).
	h.handleEditTables(tbl)
	// Reactions, by the message they are to. Facebook sends these as both V1 and V2 tables;
	// add/change carries the emoji, delete removes it (emoji "").
	for _, r := range tbl.LSUpsertReaction {
		h.notifyReaction(r.ThreadKey, r.MessageId, r.ActorId, r.Reaction)
	}
	for _, r := range tbl.LSUpdateOrInsertReactionV2 {
		h.notifyReaction(r.ThreadKey, r.MessageID, r.ReactionFBID, r.ReactionLiteral)
	}
	for _, r := range tbl.LSDeleteReaction {
		h.notifyReaction(r.ThreadKey, r.MessageId, r.ActorId, "")
	}
	for _, r := range tbl.LSDeleteReactionV2 {
		h.notifyReaction(r.ThreadKey, r.MessageID, r.ReactionFBID, "")
	}
}

// addContact records a display name and adds/updates the buddy, skipping empty ids and self.
func (h *gometaHandler) addContact(id int64, name string) {
	if id == 0 || id == h.selfID {
		return
	}
	h.mu.Lock()
	if name != "" {
		h.contactName[id] = name
	}
	h.mu.Unlock()
	h.notifyBuddy(id, name)
}

func (h *gometaHandler) handleMessage(m *table.LSInsertMessage, atts []gometaFBAttachment) {
	if m == nil || m.ThreadKey == 0 || m.IsUnsent {
		return
	}
	threadKey, senderID, offlineThreadingID, messageID, tsMs := m.ThreadKey, m.SenderId, m.OfflineThreadingId, m.MessageId, m.TimestampMs
	text, files := gometaPlainBody(m.Text, m.StickerId, atts)
	if text == "" && len(files) == 0 {
		return
	}
	// Drop the server echo of a message we just sent from here (libpurple already showed it).
	if offlineThreadingID != "" {
		if otid, err := strconv.ParseInt(offlineThreadingID, 10, 64); err == nil {
			h.mu.Lock()
			mine := h.sentOtids[otid]
			delete(h.sentOtids, otid)
			h.mu.Unlock()
			if mine {
				return
			}
		}
	}
	h.mu.Lock()
	isGroup := h.threadGroup[threadKey]
	name := h.contactName[senderID]
	partnerName := h.contactName[threadKey]
	quotedFrom := ""
	if m.ReplySourceId != "" && m.ReplyToUserId != 0 {
		quotedFrom = h.contactName[m.ReplyToUserId]
	}
	h.mu.Unlock()
	// For a 1:1 thread the ThreadKey is the conversation partner's fbid — make sure they exist
	// as a buddy even if we never saw a thread/contact row for them.
	if !isGroup {
		h.addContact(threadKey, partnerName)
	}
	msg := gometaIncoming{conv: strconv.FormatInt(threadKey, 10), who: strconv.FormatInt(senderID, 10),
		name: name, text: text, id: messageID, ts: tsMs / 1000, isGroup: isGroup, isOutgoing: senderID == h.selfID}
	// A group's news ("Anna named the group ...", "Ben left the group") comes as an admin message
	// whose text says it in words; it is a notice in the chat, nobody's message (mautrix-meta makes
	// it an m.notice).
	if m.IsAdminMessage {
		h.out().notice(msg.conv, isGroup, m.Text, msg.ts)
		return
	}
	// The message it answers, as Facebook gives it with the reply, for the quote.
	if m.ReplySourceId != "" {
		msg.quotedText = m.ReplyMessageText
		if msg.quotedText == "" {
			msg.quotedText = m.ReplySnippet
		}
		msg.quotedFrom = quotedFrom
	}
	if !msg.isOutgoing {
		h.noteUnreadFB(threadKey, tsMs) // for the read receipt when the chat is shown
	}
	if len(files) > 0 {
		gometaGo(func() { h.postPlainMedia(msg, files) })
		return
	}
	h.out().message(msg)
}

// notifyReaction shows a reaction to the message stored under messageID. emoji "" = removed.
func (h *gometaHandler) notifyReaction(threadKey int64, messageID string, actorID int64, emoji string) {
	if messageID == "" {
		return
	}
	h.out().reaction(strconv.FormatInt(threadKey, 10), messageID, strconv.FormatInt(actorID, 10), emoji)
}

func (h *gometaHandler) notifyBuddy(id int64, name string) {
	h.out().buddy(id, name)
}

func gometaNotifyBuddy(account *PurpleAccount, id int64, name string) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_buddy)}
	msg.who = C.CString(strconv.FormatInt(id, 10))
	if name != "" {
		msg.name = C.CString(name)
	}
	C.gometa_process_message(msg)
}

func (h *gometaHandler) notifyPresence(fbid int64, online bool) {
	msg := C.gometa_message_t{account: h.account, msgtype: C.char(C.gometa_message_type_presence)}
	msg.who = C.CString(strconv.FormatInt(fbid, 10))
	if online {
		msg.isOutgoing = 1
	}
	C.gometa_process_message(msg)
}

func (h *gometaHandler) notifyChat(threadKey int64, name string) { h.out().chat(threadKey, name) }

func gometaNotifyChat(account *PurpleAccount, threadKey int64, name string) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_chat)}
	msg.conv = C.CString(strconv.FormatInt(threadKey, 10))
	if name != "" {
		msg.name = C.CString(name)
	}
	C.gometa_process_message(msg)
}

func (h *gometaHandler) notifyMessage(conv, who, name, text, messageID string, ts int64, isGroup, isOutgoing bool, quotedText, quotedFrom string) {
	h.out().message(gometaIncoming{conv: conv, who: who, name: name, text: text, id: messageID, ts: ts,
		isGroup: isGroup, isOutgoing: isOutgoing, quotedText: quotedText, quotedFrom: quotedFrom})
}

func gometaNotifyMessage(account *PurpleAccount, m gometaIncoming) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_text), timestamp: C.time_t(m.ts)}
	msg.conv = C.CString(m.conv)
	msg.who = C.CString(m.who)
	if m.name != "" {
		msg.name = C.CString(m.name)
	}
	msg.text = C.CString(m.text)
	if m.id != "" {
		msg.id = C.CString(m.id)
	}
	if m.isGroup {
		msg.isGroup = 1
	}
	if m.isOutgoing {
		msg.isOutgoing = 1
	}
	C.gometa_process_message(msg)
}

// gometaNotifyTyping shows (on) or clears a 1:1 chat partner's typing.
func gometaNotifyTyping(account *PurpleAccount, who string, on bool) {
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_typing)}
	msg.who = C.CString(who)
	if on {
		msg.isOutgoing = 1
	}
	C.gometa_process_message(msg)
}

// gometaNotifySystem writes a line nobody sent into a chat (a 1:1 one, or a group's by thread key).
func gometaNotifySystem(account *PurpleAccount, conv string, isGroup bool, text string, ts int64) {
	if conv == "" || text == "" {
		return
	}
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_system), timestamp: C.time_t(ts)}
	msg.conv = C.CString(conv)
	msg.text = C.CString(text)
	if isGroup {
		msg.isGroup = 1
	}
	C.gometa_process_message(msg)
}

// ---- login ----

func gometaLogin(account *PurpleAccount, purpleUserDir, email, password, proxy string) {
	if _, ok := gometaHandlers[account]; ok {
		gometaReportError(account, "This connection already exists.", true)
		return
	}
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).
		With().Str("account", email).Timestamp().Logger()
	ctx, cancel := context.WithCancel(context.Background())
	h := &gometaHandler{
		account: account, username: email, logger: logger, ctx: ctx, cancel: cancel,
		purpleUserDir: purpleUserDir,
		threadGroup:   make(map[int64]bool), contactName: make(map[int64]string),
		sentOtids:    make(map[int64]bool),
		e2eeContacts: make(map[int64]bool),
		e2eeMsgMeta:  make(map[string]e2eeMsgInfo),
		members:      make(map[int64]map[int64]bool),
	}
	h.initParity()
	gometaHandlers[account] = h

	// A panic here, in messagix's login interpreter for one, would otherwise take the whole process
	// down, WhatsApp accounts included (mautrix-meta v0.2607.0 dereferenced a nil argument of
	// bk.action.f32.Eq during the login flow). Fail the login instead.
	go gometaRecovered(func() { h.login(email, password) }, func(r any, stack []byte) {
		h.logger.Error().Interface("panic", r).Str("stack", string(stack)).Msg("Facebook login panicked")
		h.notifyError(fmt.Sprintf("Facebook login failed: internal error (%v)", r), true)
	})
}

// login signs in, loads the inbox and connects; it runs on its own goroutine.
func (h *gometaHandler) login(email, password string) {
	ctx, logger, account := h.ctx, h.logger, h.account
	ck, err := h.obtainCookies(email, password)
	if err != nil {
		h.notifyError(fmt.Sprintf("Facebook login failed: %v", err), true)
		return
	}
	h.selfID = ck.GetUserID()
	client := messagix.NewClient(ck, logger, &messagix.Config{})
	h.client = client
	client.SetEventHandler(h.eventHandler)

	loadCtx, loadCancel := context.WithTimeout(ctx, 90*time.Second)
	user, tbl, err := client.LoadMessagesPage(loadCtx)
	loadCancel()
	if err != nil {
		// cached cookies may have expired — drop them so the next attempt re-does credentials
		gometaSetSetting(account, gometaSessionSetting, "")
		h.notifyError(fmt.Sprintf("Facebook load failed (session may have expired, try again): %v", err), true)
		return
	}
	h.logger.Info().Str("name", user.GetName()).Msg("authenticated")
	// Buddies come from the tables themselves (threads, participants, contact rows) — Messenger
	// does not hand over a bulk friend list on login (the search task 452 times out), so we
	// mirror mautrix-meta and create contacts lazily from every table we receive, here and in
	// the realtime publish stream (see parseTable + eventHandler).
	h.parseTable(tbl)
	if err := client.Connect(ctx); err != nil {
		h.notifyError(fmt.Sprintf("Facebook connect failed: %v", err), true)
		return
	}
	// Bring up the E2EE (whatsmeow) transport so encrypted threads can send/receive.
	// Non-fatal: plaintext threads keep working even if this fails.
	if err := h.connectE2EE(ck.GetUserID()); err != nil {
		h.logger.Warn().Err(err).Msg("E2EE setup failed (encrypted threads unavailable)")
	}
}

// obtainCookies returns a usable messagix session: cached cookies if present, otherwise the
// e-mail/password login. For two-factor it picks a method that takes a typed code
// (gometaPreferredMFA) and asks for the code; a login approval on another device is waited for.
// Anything else aborts with a clear message rather than looping.
func (h *gometaHandler) obtainCookies(email, password string) (*cookies.Cookies, error) {
	// 1) cached session
	if blob := gometaGetSetting(h.account, gometaSessionSetting); blob != "" {
		ck := &cookies.Cookies{Platform: types.Facebook}
		if err := json.Unmarshal([]byte(blob), ck); err == nil && ck.IsLoggedIn() {
			h.logger.Info().Msg("using cached Facebook session")
			return ck, nil
		}
	}
	if email == "" || password == "" {
		return nil, fmt.Errorf("enter your Facebook email and password")
	}
	// Bound the whole login so an unapproved login-approval can't wait forever.
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Minute)
	defer cancel()

	loginClient := messagix.NewClient(&cookies.Cookies{Platform: types.MessengerLiteIOS}, h.logger, &messagix.Config{})
	var userInput map[string]string
	credsSent := false
	for iter := 0; iter < 40; iter++ { // hard loop guard
		step, newCookies, err := loginClient.MessengerLite.DoLoginSteps(ctx, userInput)
		if err != nil {
			return nil, err
		}
		if step == nil { // login complete
			if newCookies == nil {
				return nil, fmt.Errorf("login returned no session")
			}
			ck := &cookies.Cookies{Platform: types.Facebook}
			ck.UpdateValues(newCookies.GetAll())
			if b, e := json.Marshal(ck); e == nil {
				gometaSetSetting(h.account, gometaSessionSetting, string(b))
			}
			return ck, nil
		}
		switch step.Type {
		case bridgev2.LoginStepTypeDisplayAndWait:
			// Login approval: Facebook sent a notification to your other device. The next
			// DoLoginSteps call blocks (polling on a timer) until you approve — not a busy loop.
			h.logger.Info().Str("info", step.Instructions).Msg("waiting for login approval on your other device")
			userInput = nil
		case bridgev2.LoginStepTypeUserInput:
			ui, err := h.fillLoginStep(step, email, password, &credsSent)
			if err != nil {
				return nil, err // can't fill (code method / rejected creds) -> abort, do not loop
			}
			userInput = ui
		default:
			return nil, fmt.Errorf("unsupported Facebook login step %q", step.Type)
		}
	}
	return nil, fmt.Errorf("Facebook login did not complete")
}

// fillLoginStep builds the submit map: e-mail/password from the account, the 2FA method, and the
// 2FA code the user types. Returns an error (which aborts the login) for any other kind of input,
// so we never spin on a prompt that can't be answered.
func (h *gometaHandler) fillLoginStep(step *bridgev2.LoginStep, email, password string, credsSent *bool) (map[string]string, error) {
	out := map[string]string{}
	if step.UserInputParams == nil {
		return out, nil
	}
	for _, f := range step.UserInputParams.Fields {
		switch f.Type {
		case bridgev2.LoginInputFieldTypeEmail, bridgev2.LoginInputFieldTypeUsername, bridgev2.LoginInputFieldTypePhoneNumber:
			if *credsSent {
				return nil, fmt.Errorf("Facebook rejected the email/password")
			}
			out[f.ID] = email
		case bridgev2.LoginInputFieldTypePassword:
			out[f.ID] = password
			*credsSent = true
		case bridgev2.LoginInputFieldTypeSelect:
			// 2FA method picker: choose a code method (Authentication app / Text message);
			// the code is then asked for (see the 2FACode case).
			choice := gometaPreferredMFA(f.Options)
			if choice == "" {
				return nil, fmt.Errorf("this Facebook account offers no code-based two-factor method we can use (%v) — enable Authentication app or Text message in Facebook Security settings", f.Options)
			}
			h.logger.Info().Str("mfa", choice).Msg("selecting 2FA method")
			out[f.ID] = choice
		case bridgev2.LoginInputFieldType2FACode:
			// Facebook wants the login/2FA code; block until the user enters it (promptInput).
			h.logger.Info().Str("prompt", step.Instructions).Msg("waiting for the 2FA code")
			code := strings.TrimSpace(h.promptInput(step.Instructions))
			if code == "" {
				return nil, fmt.Errorf("no Facebook 2FA code was entered")
			}
			out[f.ID] = code
		default:
			return nil, fmt.Errorf("this Facebook login needs a typed %q, which isn't supported yet", f.Type)
		}
	}
	return out, nil
}

// gometaPreferredMFA picks a 2FA method we can complete by typing a code. Code methods are preferred in this order. "Notification on another device" (AFAD)
// is deliberately NOT selected: Meta rejects it in this messagix flow (the afad_state.async RPC
// returns an error, which older mautrix-meta even crashes parsing), so it can never complete.
func gometaPreferredMFA(options []string) string {
	for _, want := range []string{"Authentication app", "Text message", "WhatsApp"} {
		for _, o := range options {
			if o == want {
				return o
			}
		}
	}
	return ""
}
