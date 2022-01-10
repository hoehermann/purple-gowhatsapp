package main

/*
#include "constants.h"
*/
import "C"

import (
	"go.mau.fi/whatsmeow/types"
	"time"
)

/*
 * Holds ID and sender of a received message so the receipt can be sent later.
 */
var deferredReceipts = make(map[types.JID]map[types.JID][]types.MessageID)

/*
 * Puts ID and sender of a received message in the store.
 *
 * Except if the purple setting is "never send receipts". Then it does nothing.
 */
func (handler *Handler) mark_read_defer(id types.MessageID, chat types.JID, sender types.JID) {
	immediately := C.GoString(C.GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY)
	never := C.GoString(C.GOWHATSAPP_SEND_RECEIPT_CHOICE_NEVER)
	setting := purple_get_string(handler.account, C.GOWHATSAPP_SEND_RECEIPT_OPTION, immediately)
	if never == setting {
		return
	}

	// append to map of map of ids
	// I really could go for a python defaultdict here
	senders, ok := deferredReceipts[chat]
	if !ok {
		// first sender in this chat
		senders = make(map[types.JID][]types.MessageID)
		ids := []types.MessageID{id}
		senders[sender] = ids
	} else {
		// chat already has senders
		ids, ok := senders[sender]
		if !ok {
			// first id for this sender
			ids = []types.MessageID{id}
		} else {
			ids = append(ids, id)
		}
		senders[sender] = ids
	}
	deferredReceipts[chat] = senders
}

/*
 * In case the purple setting is matching,
 * ths will send all receipts stored for the given chat.
 * mark_read_defer(…) should have been called before.
 */
func (handler *Handler) mark_read_if(chat types.JID, choice *C.char) {
	default_setting := C.GoString(C.GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY)
	setting := purple_get_string(handler.account, C.GOWHATSAPP_SEND_RECEIPT_OPTION, default_setting)
	if C.GoString(choice) == setting {
		handler.log.Infof("Sending read receipt due to setting %s...", C.GoString(choice))
		handler.mark_read_unconditionally(chat)
	}
}

/*
 * Shorthand
 */
func (handler *Handler) mark_read_if_on_answer(chat types.JID) {
	handler.mark_read_if(chat, C.GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_ANSWER)
}

/*
 * Shorthand
 */
func (handler *Handler) mark_read_if_on_receival(chat types.JID) {
	handler.mark_read_if(chat, C.GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY)
}

/*
 * In case the purple setting is "send receipt when interacting with conversation",
 * ths will send all receipts stored for the given chat.
 * mark_read_defer(…) should have been called before.
 */
func (handler *Handler) mark_read_conversation(chat string) {
	recipient, err := parseJID(chat)
	if err != nil {
		// fail silently
	} else {
		handler.mark_read_if(recipient, C.GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_INTERACT)
	}
}

/*
 * This unconditionally sends all receipts stored in regard to the given chat.
 * mark_read_defer(…) should have been called before.
 */
func (handler *Handler) mark_read_unconditionally(chat types.JID) {
	senders, ok := deferredReceipts[chat]
	if ok {
		for sender, ids := range senders {
			handler.mark_read(ids, chat, sender)
		}
	}
}

/*
 * This actually sends the receipts.
 */
func (handler *Handler) mark_read(ids []types.MessageID, chat types.JID, sender types.JID) {
	handler.log.Infof("Sending read receipt to %s...", sender.ToNonAD().String())
	err := handler.client.MarkRead(ids, time.Now(), chat, sender)
	if err != nil {
		handler.log.Warnf("Sending read receipt failed: %#v", err)
	}
}
