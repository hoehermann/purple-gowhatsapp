package main

/*
#include "bridge.h"
*/
import "C"

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

/*
 * Forwards an incoming receipt to the purple part, one call per message ID
 * (a single event may confirm many messages, and their order is platform
 * dependent, so no order is implied).
 *
 * Only the receipt types a conversation window can meaningfully show are
 * forwarded: delivered, read and read-self. Retry and played receipts are
 * of no use to a chat UI and are dropped here.
 */
func (handler *Handler) handle_receipt(evt *events.Receipt) {
	var receiptType C.char
	switch evt.Type {
	case types.ReceiptTypeDelivered:
		receiptType = C.char(C.gowhatsapp_receipt_type_delivered)
	case types.ReceiptTypeRead:
		receiptType = C.char(C.gowhatsapp_receipt_type_read)
	case types.ReceiptTypeReadSelf:
		receiptType = C.char(C.gowhatsapp_receipt_type_read_self)
	default:
		handler.log.Debugf("Ignoring receipt of type %s from %s.", evt.Type, evt.SourceString())
		return
	}
	// same LID to phone number mapping the message path applies, else the JIDs
	// in a receipt would not match the conversations they belong to
	chat := handler.lidToPn(evt.Chat, "handling receipt chat")
	sender := handler.lidToPn(evt.Sender, "handling receipt sender")
	for _, id := range evt.MessageIDs {
		purple_receipt(handler.account, chat.ToNonAD().String(), sender.ToNonAD().String(), evt.IsGroup, evt.Timestamp, receiptType, id)
	}
}

/*
 * Reports this instance's own message as sent (the first tick): the server has
 * accepted it and assigned the ID all future receipts will refer to. Without
 * this a consumer could never match a delivered or read receipt to the message
 * it confirms.
 */
func (handler *Handler) receipt_sent(recipient types.JID, isGroup bool, send_response whatsmeow.SendResponse) {
	ownJid := handler.client.Store.ID.ToNonAD().String()
	purple_receipt(handler.account, recipient.ToNonAD().String(), ownJid, isGroup, send_response.Timestamp, C.char(C.gowhatsapp_receipt_type_sent), send_response.ID)
}
