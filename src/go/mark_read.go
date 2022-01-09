package main

/*
#include "constants.h"
*/
import "C"

import (
	"go.mau.fi/whatsmeow/types"
	"time"
)

func (handler *Handler) mark_read_immediately(id types.MessageID, chat types.JID, sender types.JID) {
	immediately := C.GoString(C.GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY)
	if immediately == purple_get_string(handler.username, C.GOWHATSAPP_SEND_RECEIPT_OPTION, C.GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY) {
		handler.mark_read([]types.MessageID{id}, chat, sender)
	}
}

func (handler *Handler) mark_read(ids []types.MessageID, chat types.JID, sender types.JID) {
	handler.log.Infof("Sending read receipt to %s...", sender.ToNonAD().String())
	err := handler.client.MarkRead(ids, time.Now(), chat, sender)
	if err != nil {
		handler.log.Warnf("Sending read receipt failed: %#v", err)
	}
}
