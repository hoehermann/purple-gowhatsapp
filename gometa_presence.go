//go:build gometa

package main

// Typing, both ways, and read receipts sent, for Facebook. Encrypted chats go over the
// whatsmeow client, as mautrix-meta's HandleMatrixTyping and HandleMatrixReadReceipt do; plain
// chats over messagix tasks (UpdatePresenceTask, ThreadMarkReadTask). Typing is for 1:1 chats only:
// libpurple 2 has no typing state for a group chat.
//
// An encrypted chat is read on two networks: the whatsmeow receipt goes to the sender's devices,
// and Messenger's own apps take a chat's read state from its Facebook ("hybrid") thread, which
// gets a ThreadMarkReadTask in sync group 95 under its Facebook thread key, as mautrix-meta does
// since its "mark hybrid threads as read on Lightspeed" (mautrix/meta#474).

/*
#include "gometabridge.h"
*/
import "C"

import (
	"context"
	"strconv"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// gometaMaxUnread bounds the encrypted messages kept per chat for its next read receipt.
const gometaMaxUnread = 50

// gometaUnread is an encrypted message not yet marked read: the receipt names its chat and sender.
type gometaUnread struct {
	id           waTypes.MessageID
	chat, sender waTypes.JID
}

// gometaParity is the handler's state for typing and receipts, kept by thread key.
type gometaParity struct {
	threadType map[int64]table.ThreadType // from the thread rows, for the typing task
	e2eeChat   map[int64]waTypes.JID      // an encrypted chat's JID, as its messages came from
	unreadE2EE map[int64][]gometaUnread   // encrypted messages to mark read
	unreadFB   map[int64]int64            // a plain chat's newest unread message time (ms)
	// an encrypted chat's newest message time (ms), for its Facebook thread's read state
	readE2EE map[int64]int64
	// an encrypted chat's Facebook thread key, by the chat's WhatsApp-side id, where they differ
	// (LSVerifyHybridThreadExists, LSUpdateThreadAuthorityAndMappingWithOTIDFromJID)
	hybridKey map[int64]int64
}

// gometaHybridSyncGroup is the sync group of an encrypted chat's Facebook thread (mautrix-meta
// marks such a thread read, accepts and deletes it there).
const gometaHybridSyncGroup = 95

func (h *gometaHandler) initParity() {
	h.parity = gometaParity{
		threadType: make(map[int64]table.ThreadType),
		e2eeChat:   make(map[int64]waTypes.JID),
		unreadE2EE: make(map[int64][]gometaUnread),
		unreadFB:   make(map[int64]int64),
		readE2EE:   make(map[int64]int64),
		hybridKey:  make(map[int64]int64),
	}
}

//export gometa_go_send_typing
func gometa_go_send_typing(account *PurpleAccount, who *C.char, typing C.int) {
	if h, ok := gometaHandlers[account]; ok {
		go h.sendTyping(C.GoString(who), typing != 0)
	}
}

// The UI has shown this chat (a thread key), so what came in it has been read. Raised through
// libpurple's "conversation-updated" (glue/gometa_init.c).
//
//export gometa_go_mark_read
func gometa_go_mark_read(account *PurpleAccount, who *C.char) {
	if h, ok := gometaHandlers[account]; ok {
		go h.markRead(C.GoString(who))
	}
}

// e2eeJID is the JID an encrypted chat is reached at: the one its messages came from, else a 1:1
// Messenger chat's.
func (h *gometaHandler) e2eeJID(threadKey int64) waTypes.JID {
	h.mu.Lock()
	jid, ok := h.parity.e2eeChat[threadKey]
	h.mu.Unlock()
	if ok {
		return jid
	}
	return waTypes.JID{User: strconv.FormatInt(threadKey, 10), Server: waTypes.MessengerServer}
}

// sendTyping tells a chat we are typing (on) or stopped. who is the thread key; anything else
// (the "Facebook" sign-in chat) is left alone.
func (h *gometaHandler) sendTyping(who string, on bool) {
	threadKey, err := strconv.ParseInt(who, 10, 64)
	if err != nil || threadKey == 0 {
		return
	}
	h.mu.Lock()
	isE2EE := h.e2eeContacts[threadKey]
	isGroup := h.threadGroup[threadKey]
	threadType := h.parity.threadType[threadKey]
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	defer cancel()
	if isE2EE {
		err = h.net().chatPresence(ctx, h.e2eeJID(threadKey), on)
	} else {
		task := &socket.UpdatePresenceTask{ThreadKey: threadKey, SyncGroup: 1, ThreadType: int64(threadType)}
		if isGroup {
			task.IsGroupThread = 1
		}
		if threadType == 0 {
			task.ThreadType = int64(table.ONE_TO_ONE)
			if isGroup {
				task.ThreadType = int64(table.GROUP_THREAD)
			}
		}
		if on {
			task.IsTyping = 1
		}
		err = h.net().fbTask(ctx, task, true)
	}
	if err != nil {
		h.logger.Debug().Err(err).Bool("e2ee", isE2EE).Msg("typing not sent")
	}
}

// gometaE2EEReadMs is how far an encrypted message's Facebook thread is read once it is: its time in
// ms, to the end of its second when it has none finer (whatsmeow's come in whole seconds, and the
// thread's own time for it may be later in that second), as mautrix-meta rounds it.
func gometaE2EEReadMs(ts time.Time) int64 {
	if ts.IsZero() {
		return 0
	}
	if ts.Nanosecond() == 0 {
		ts = ts.Add(999 * time.Millisecond)
	}
	return ts.UnixMilli()
}

// noteHybridThreads keeps which Facebook thread an encrypted chat is, where the keys differ.
func (h *gometaHandler) noteHybridThreads(tbl *table.LSTable) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.parity.hybridKey == nil {
		return
	}
	for _, m := range tbl.LSVerifyHybridThreadExists {
		if m.ThreadKey != 0 && m.ThreadJID != 0 {
			h.parity.hybridKey[m.ThreadJID] = m.ThreadKey
		}
	}
	for _, m := range tbl.LSUpdateThreadAuthorityAndMappingWithOTIDFromJID {
		if m.ThreadKey != 0 && m.ThreadJID != 0 {
			h.parity.hybridKey[m.ThreadJID] = m.ThreadKey
		}
	}
}

// fbThreadOfE2EE is the Facebook thread key of an encrypted chat: the one Facebook named for it,
// else, for a 1:1 chat, the chat's own (a 1:1 thread's key is the partner's fbid on both sides).
// 0 for an encrypted group Facebook named none for.
func (h *gometaHandler) fbThreadOfE2EE(threadKey int64, chat waTypes.JID) int64 {
	h.mu.Lock()
	key, ok := h.parity.hybridKey[threadKey]
	h.mu.Unlock()
	if ok {
		return key
	}
	if chat.Server == waTypes.GroupServer {
		return 0
	}
	return threadKey
}

// noteUnreadE2EE keeps a received encrypted message for the chat's next read receipt: the
// message itself when someone else sent it, and its time for the Facebook thread either way (one
// of ours from another device was read there too).
func (h *gometaHandler) noteUnreadE2EE(threadKey int64, info waTypes.MessageInfo) {
	if threadKey == 0 || info.ID == "" {
		return
	}
	h.mu.Lock()
	h.parity.e2eeChat[threadKey] = info.Chat.ToNonAD()
	if ms := gometaE2EEReadMs(info.Timestamp); ms > h.parity.readE2EE[threadKey] {
		h.parity.readE2EE[threadKey] = ms
	}
	if info.IsFromMe {
		h.mu.Unlock()
		return
	}
	list := append(h.parity.unreadE2EE[threadKey], gometaUnread{id: info.ID, chat: info.Chat.ToNonAD(), sender: info.Sender.ToNonAD()})
	if len(list) > gometaMaxUnread {
		list = list[len(list)-gometaMaxUnread:]
	}
	h.parity.unreadE2EE[threadKey] = list
	h.mu.Unlock()
}

// noteUnreadFB keeps a received plain message's time for the chat's next read receipt.
func (h *gometaHandler) noteUnreadFB(threadKey, tsMs int64) {
	if threadKey == 0 || tsMs == 0 {
		return
	}
	h.mu.Lock()
	if tsMs > h.parity.unreadFB[threadKey] {
		h.parity.unreadFB[threadKey] = tsMs
	}
	h.mu.Unlock()
}

// markRead sends read receipts for what came in a chat since its last ones: in an encrypted chat
// for each message, one receipt per sender (a group's receipt names its sender); in a plain chat
// one up to the newest message's time, as mautrix-meta's HandleMatrixReadReceipt does.
func (h *gometaHandler) markRead(who string) {
	threadKey, err := strconv.ParseInt(who, 10, 64)
	if err != nil || threadKey == 0 {
		return
	}
	h.mu.Lock()
	unread := h.parity.unreadE2EE[threadKey]
	delete(h.parity.unreadE2EE, threadKey)
	watermark := h.parity.unreadFB[threadKey]
	delete(h.parity.unreadFB, threadKey)
	e2eeWatermark := h.parity.readE2EE[threadKey]
	delete(h.parity.readE2EE, threadKey)
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()
	type key struct{ chat, sender waTypes.JID }
	var order []key
	bySender := make(map[key][]waTypes.MessageID)
	for _, u := range unread {
		k := key{u.chat, u.sender}
		if _, seen := bySender[k]; !seen {
			order = append(order, k)
		}
		bySender[k] = append(bySender[k], u.id)
	}
	for _, k := range order {
		if err := h.net().markRead(ctx, bySender[k], time.Now(), k.chat, k.sender); err != nil {
			h.logger.Debug().Err(err).Msg("encrypted read receipt not sent")
		}
	}
	if watermark != 0 {
		task := &socket.ThreadMarkReadTask{ThreadId: threadKey, LastReadWatermarkTs: watermark, SyncGroup: 1}
		if err := h.net().fbTask(ctx, task, false); err != nil {
			h.logger.Debug().Err(err).Msg("read receipt not sent")
		}
	}
	// The encrypted chat's Facebook thread, read up to its newest message.
	if e2eeWatermark != 0 {
		fbKey := h.fbThreadOfE2EE(threadKey, h.e2eeJID(threadKey))
		if fbKey == 0 {
			return
		}
		task := &socket.ThreadMarkReadTask{ThreadId: fbKey, LastReadWatermarkTs: e2eeWatermark, SyncGroup: gometaHybridSyncGroup}
		if err := h.net().fbTask(ctx, task, false); err != nil {
			h.logger.Debug().Err(err).Msg("encrypted chat's Facebook thread not marked read")
		}
	}
}

// handleTypingTable shows a plain 1:1 chat's partner typing (LSUpdateTypingIndicator).
func (h *gometaHandler) handleTypingTable(rows []*table.LSUpdateTypingIndicator) {
	for _, t := range rows {
		if t.ThreadKey == 0 || t.SenderId == h.selfID {
			continue
		}
		h.mu.Lock()
		isGroup := h.threadGroup[t.ThreadKey]
		h.mu.Unlock()
		if isGroup {
			continue
		}
		h.out().typing(strconv.FormatInt(t.ThreadKey, 10), t.IsTyping)
	}
}

// handleChatPresence shows an encrypted 1:1 chat's partner typing.
func (h *gometaHandler) handleChatPresence(evt *events.ChatPresence) {
	if evt.IsFromMe || evt.Chat.Server != waTypes.MessengerServer || evt.Chat.User == "" {
		return
	}
	h.out().typing(evt.Chat.User, evt.State == waTypes.ChatPresenceComposing)
}

// announceAvailable marks us online on the encrypted transport once it is connected: WhatsApp,
// which carries Facebook's encrypted chats, sends typing only to a user who is online (as
// mautrix-meta notes in HandleMatrixViewingChat).
func (h *gometaHandler) announceAvailable(e2ee *whatsmeow.Client) {
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()
	if err := e2ee.SendPresence(ctx, waTypes.PresenceAvailable); err != nil {
		h.logger.Debug().Err(err).Msg("E2EE presence not sent")
	}
}
