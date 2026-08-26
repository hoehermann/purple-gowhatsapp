package main

/*
#include "constants.h"
#include "bridge.h"
*/
import "C"

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
)

/*
 * Asks the primary device for the messages preceding the newest message this
 * client knows of the given conversation. The phone answers asynchronously
 * (and only while WhatsApp is running there) with an events.HistorySync of
 * type ON_DEMAND, handled by handle_history_sync below.
 *
 * The request needs a real anchor message (chat, id, sender and timestamp);
 * the message cache provides it. Without a cached message for the chat the
 * request points at the present moment and names no message.
 */
func (handler *Handler) request_history(chatJid string, count int) {
	if count <= 0 || chatJid == "" {
		return
	}
	// parseJID, not types.ParseJID: purple hands over conversation names like
	// "+4915112345678", which types.ParseJID would silently read as a bare
	// server name and the phone would never answer
	chat, err := parseJID(chatJid)
	if err != nil {
		return
	}
	if handler.client.Store.ID == nil {
		// not linked to a phone yet, so there is nobody to ask
		return
	}
	if last, ok := handler.historyRequests[chatJid]; ok && time.Since(last) < time.Minute {
		// a conversation being re-created in quick succession is not a new request
		return
	}
	for jid, when := range handler.historyRequests {
		// an entry past its minute holds nothing back and need not be kept
		if time.Since(when) > time.Minute {
			delete(handler.historyRequests, jid)
		}
	}
	handler.historyRequests[chatJid] = time.Now()

	/* The request points at a message and asks for what came before it. The
	 * cache supplies that anchor where it can; a conversation nobody has spoken
	 * in since this client started is not in the cache, and rather than give up
	 * there the request points at the present moment and names no message. The
	 * phone decides whether it will answer that. */
	info := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:    chat,
			IsGroup: chat.Server == types.GroupServer,
		},
		Timestamp: time.Now(),
	}
	if anchor := handler.newest_cached_anchor(chat); anchor != nil {
		/* Everything but the conversation itself comes from the anchor. The
		 * conversation stays the one that was parsed and asked about: the cache
		 * records a chat as the message path left it, which for a status
		 * broadcast is a particular device rather than the conversation, and
		 * the request goes out with no device suffix trimmed off it. */
		ownJid := handler.client.Store.ID.ToNonAD()
		info.Sender = anchor.sender
		info.IsFromMe = anchor.sender.ToNonAD() == ownJid
		info.ID = anchor.id
		info.Timestamp = anchor.timestamp
		handler.log.Infof("Requesting history for %s before message %s of %s.", chatJid, anchor.id, anchor.timestamp)
	} else {
		handler.log.Infof("Requesting history for %s with no anchor message; the cache knows none.", chatJid)
	}
	request := handler.client.BuildHistorySyncRequest(&info, count)
	go func() {
		_, err := handler.client.SendPeerMessage(context.Background(), request)
		if err != nil {
			handler.log.Warnf("Unable to request history for %s: %v", chatJid, err)
		}
	}()
}

/* What request_history takes away from the newest cached message: scalars
 * only, copied out under the lock. request_history runs on purple's thread,
 * so a pointer into the slice would race the next append and trim, and a
 * whole CachedMessage must never be copied by value, its embedded protobuf
 * message is marked do-not-copy (the same reason add_to_cache copies
 * field-by-field). */
type cachedAnchor struct {
	id        string
	sender    types.JID
	timestamp time.Time
}

func (handler *Handler) newest_cached_anchor(chat types.JID) *cachedAnchor {
	handler.cacheMutex.Lock()
	defer handler.cacheMutex.Unlock()
	// the cache appends chronologically, so the last match is the newest
	for i := len(handler.cachedMessages) - 1; i >= 0; i-- {
		if handler.cachedMessages[i].Chat.ToNonAD() == chat.ToNonAD() {
			return &cachedAnchor{
				id:        handler.cachedMessages[i].ID,
				sender:    handler.cachedMessages[i].Sender,
				timestamp: handler.cachedMessages[i].Timestamp,
			}
		}
	}
	return nil
}

/* The lines the cache can add beyond the phone's answer: messages of this
 * conversation from notBefore on, excluding ids already shown. Built under
 * the lock, reading each protobuf message in place rather than copying it. */
func (handler *Handler) history_entries_from_cache(chat types.JID, seen map[string]bool, notBefore time.Time) []historyEntry {
	handler.cacheMutex.Lock()
	defer handler.cacheMutex.Unlock()
	entries := []historyEntry{}
	for i := range handler.cachedMessages {
		cached := &handler.cachedMessages[i]
		if cached.Chat.ToNonAD() != chat.ToNonAD() {
			continue
		}
		if seen[cached.ID] || cached.Timestamp.Before(notBefore) {
			continue
		}
		text := history_text(&cached.Message)
		if text == "" {
			continue
		}
		entries = append(entries, historyEntry{
			id:        cached.ID,
			sender:    cached.Sender,
			timestamp: cached.Timestamp,
			text:      text,
		})
	}
	return entries
}

// one conversation line reconstructed from the phone's history or the cache
type historyEntry struct {
	id        string
	sender    types.JID
	timestamp time.Time
	text      string
	pushName  string
}

/*
 * The phone's answer to request_history: conversations full of WebMessageInfo.
 * Each is turned back into a displayable line, merged with what the cache
 * knows (the anchor and anything newer), sorted chronologically and displayed
 * as history: delayed messages carrying their ids.
 */
func (handler *Handler) handle_history_sync(data *waHistorySync.HistorySync) {
	if handler.client.Store.ID == nil {
		return
	}
	// the phone answers asynchronously or not at all, so say when it did
	handler.log.Infof("History sync answer for %d conversation(s).", len(data.GetConversations()))
	ownJid := handler.client.Store.ID.ToNonAD()
	for _, conversation := range data.GetConversations() {
		chat, err := types.ParseJID(conversation.GetID())
		if err != nil {
			handler.log.Warnf("History sync for unparseable conversation %s: %v", conversation.GetID(), err)
			continue
		}
		chat = handler.lidToPn(chat, "handling history conversation")
		isGroup := chat.Server == types.GroupServer

		entries := []historyEntry{}
		seen := map[string]bool{}
		var newestFromPhone time.Time
		var unshowable int
		var reactionsSetAside int
		for _, historyMsg := range conversation.GetMessages() {
			webMsg := historyMsg.GetMessage()
			evt, err := handler.client.ParseWebMessage(chat, webMsg)
			if err != nil {
				handler.log.Warnf("Dropping unparseable history message in %s: %v", chat, err)
				continue
			}
			/* Our own messages name their author in whatever identity the phone
			 * recorded at the time, which for some is the hidden lid rather than
			 * the phone number, and no mapping brings that one home: it is our
			 * own, and the store maps other people's. A room would then seat us
			 * twice, under two names, one of them a number nobody recognises. */
			sender := ownJid
			if !evt.Info.IsFromMe {
				sender = handler.lidToPn(evt.Info.Sender, "handling history sender")
			}

			/* A reaction is a message in its own right here, naming the message
			 * it belongs to. It is not a line of conversation, and the textual
			 * rendering has nothing to hang it on, so it is set aside. */
			if evt.Message.GetReactionMessage() != nil {
				reactionsSetAside++
				continue
			}

			text := history_text(evt.Message)
			if text == "" {
				unshowable++
				continue
			}
			/* An edited message is parsed under the id of the message it edits,
			 * which is the right id to carry but would put two lines on the page
			 * claiming to be the same message. The later one wins. */
			if seen[evt.Info.ID] {
				for i := range entries {
					if entries[i].id == evt.Info.ID {
						entries = append(entries[:i], entries[i+1:]...)
						break
					}
				}
			}
			entries = append(entries, historyEntry{
				id:        evt.Info.ID,
				sender:    sender,
				timestamp: evt.Info.Timestamp,
				text:      text,
				pushName:  evt.Info.PushName,
			})
			seen[evt.Info.ID] = true
			if evt.Info.Timestamp.After(newestFromPhone) {
				newestFromPhone = evt.Info.Timestamp
			}
		}

		handler.log.Infof("History for %s: %d in the answer, %d to show, %d reaction(s) set aside, %d with nothing to show.",
			chat, len(conversation.GetMessages()), len(entries), reactionsSetAside, unshowable)

		/* The phone answers with what lies before the anchor and stops there, so
		 * the anchor itself and anything after it is missing from its answer.
		 * The cache fills exactly that gap, and nothing else: an older message
		 * the cache happens to hold is one the phone chose not to send, and it
		 * is not this client's place to overrule that. An answer that brought no
		 * messages marks no gap, so the cache stays out of it entirely. */
		if len(entries) > 0 {
			entries = append(entries, handler.history_entries_from_cache(chat, seen, newestFromPhone)...)
		}

		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].timestamp.Before(entries[j].timestamp)
		})

		// the user asked for a number of messages, so show that many, the newest
		if wanted := purple_get_int(handler.account, C.GOWHATSAPP_FETCH_HISTORY_OPTION, 0); wanted > 0 && len(entries) > wanted {
			entries = entries[len(entries)-wanted:]
		}

		for _, entry := range entries {
			var pushName *string
			if entry.pushName != "" {
				name := entry.pushName
				pushName = &name
			}
			purple_display_history_message(handler.account, chat.ToNonAD().String(), isGroup, entry.sender.ToNonAD().String(), pushName, entry.timestamp, entry.text, entry.id)
		}
	}
}

/*
 * The line of text a historical message displays as. Text carries over as-is;
 * media becomes a short placeholder (historical attachments are not
 * downloaded), keeping its caption if it had one.
 */
func history_text(message *waE2E.Message) string {
	if message == nil {
		return ""
	}
	if text := message.GetConversation(); text != "" {
		return text
	}
	if text := message.GetExtendedTextMessage().GetText(); text != "" {
		return text
	}
	if m := message.GetImageMessage(); m != nil {
		return history_placeholder("an image", m.GetCaption())
	}
	if m := message.GetVideoMessage(); m != nil {
		return history_placeholder("a video", m.GetCaption())
	}
	if message.GetPtvMessage() != nil {
		return "[Sent a video message.]"
	}
	if message.GetAudioMessage() != nil {
		return "[Sent a voice message.]"
	}
	if message.GetStickerMessage() != nil {
		return "[Sent a sticker.]"
	}
	if m := message.GetDocumentMessage(); m != nil {
		return history_placeholder("a file: "+m.GetFileName(), m.GetCaption())
	}
	if message.GetContactMessage() != nil || message.GetContactsArrayMessage() != nil {
		return "[Sent a contact.]"
	}
	/* Kinds that carry no text but enough to be named, so that a conversation
	 * replayed from the phone has no silent holes in it. */
	if m := message.GetLocationMessage(); m != nil {
		label := m.GetName()
		if label == "" {
			label = m.GetAddress()
		}
		if label == "" {
			label = "Shared location"
		}
		return fmt.Sprintf("%s: https://www.openstreetmap.org/?mlat=%f&mlon=%f", label,
			m.GetDegreesLatitude(), m.GetDegreesLongitude())
	}
	if m := message.GetLiveLocationMessage(); m != nil {
		return fmt.Sprintf("Shared live location (last position: https://www.openstreetmap.org/?mlat=%f&mlon=%f)",
			m.GetDegreesLatitude(), m.GetDegreesLongitude())
	}
	if m := GetAnyPollCreationMessage(message); m != nil {
		return "[POLL] " + m.GetName()
	}
	/* Everything else, group events and calls and votes among them, has nothing
	 * to say as a line of conversation and is left out on purpose. */
	return ""
}

func history_placeholder(what string, caption string) string {
	if caption != "" {
		return "[Sent " + what + ".] " + caption
	}
	return "[Sent " + what + ".]"
}
