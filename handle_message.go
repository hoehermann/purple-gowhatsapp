package main

/*
#include "constants.h"
*/
import "C"

import (
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func (handler *Handler) handle_message(message *waE2E.Message, info types.MessageInfo) {
	//handler.log.Infof("message: %#v", message)
	text := ""
	if info.MessageSource.Chat == types.StatusBroadcastJID {
		if purple_get_bool(handler.account, C.GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, false) {
			// some people find status broadcasts annoying
			handler.log.Warnf("Ignoring status broadcast.")
			return
		} else {
			// the protocol implements status broadcasts in the form of a group
			// we just treat those messages as if they were direct messages
			info.MessageSource.Chat = info.MessageSource.Sender
			info.MessageSource.IsGroup = false
			text = "[STATUS] "
		}
	}
	if handler.blocklist != nil {
		// TODO find out whether locally checking the blocklist is actually necessary or if WhatsApp servers do the filtering for us
		for _, blockedJID := range handler.blocklist.JIDs {
			if blockedJID.ToNonAD() == info.MessageSource.Sender.ToNonAD() {
				handler.log.Infof("Ignoring message from %s since they are on the blocklist.", info.MessageSource.Sender.ToNonAD().String())
			}
		}
	}
	info.MessageSource.Chat = handler.lidToPn(info.MessageSource.Chat, "handling message chat")
	info.MessageSource.Sender = handler.lidToPn(info.MessageSource.Sender, "handling message sender")
	isEdit := false
	{
		if pm := message.GetProtocolMessage(); pm != nil {
			if em := pm.GetEditedMessage(); em != nil {
				message = em
				isEdit = true
			}
		}
	}
	text += message.GetConversation()
	{
		etm := message.ExtendedTextMessage
		if etm != nil {
			// message containing quote or link to group
			// link messages have message.Conversation set to nil anyway
			// it should be safe to overwrite here
			// quoted message repeats the text
			ci := etm.ContextInfo
			if ci != nil {
				cm := ci.QuotedMessage
				if cm != nil && cm.Conversation != nil {
					quotelines := strings.Split(*cm.Conversation, "\n")
					text = "> " + strings.Join(quotelines, "\n> ") + "\n"
				}
			}
			if etm.Text != nil {
				text += *etm.Text
			}
		}

	}
	{
		rm := message.GetReactionMessage()
		if rm != nil && rm.Text != nil && rm.Key != nil && rm.Key.ID != nil {
			quote := ""
			cached_message := handler.lookup_cached_message_by_id(rm.Key.GetID())
			if cached_message != nil {
				text := cached_message.Message.GetConversation() // TODO: check if this works for quoting messages via ExtendedTextMessage, too
				quote = fmt.Sprintf("message \"%.50s\" from %s", text, cached_message.Timestamp.Format(time.RFC822))
				// TODO: add elipis to indicate message truncation
			}
			if quote == "" {
				quote = fmt.Sprintf("unknown message with ID %s", rm.Key.GetID())
			}
			if *rm.Text == "" {
				text += fmt.Sprintf("removed their reaction to %s.", quote)
			} else {
				text += fmt.Sprintf("reacted with %s to %s.", *rm.Text, quote)
			}
		}
	}
	if message.GetPollCreationMessage() != nil || message.GetPollCreationMessageV2() != nil || message.GetPollCreationMessageV3() != nil {
		text = "created a poll, but this plug-in cannot display polls."
		// TODO: display poll content
		// TODO: also use GetPollUpdateMessage()
	}
	if text == "" {
		handler.log.Warnf("Received a message without any text.")
	} else {
		if isEdit {
			text = "[EDIT] " + text
		}
		// note: info.PushName always denotes the sender (not the chat)
		purple_display_text_message(handler.account, info.MessageSource.Chat.ToNonAD().String(), info.MessageSource.IsGroup, false, info.MessageSource.Sender.ToNonAD().String(), &info.PushName, info.Timestamp, text, &info.ID)
	}
	if !isEdit { // edited messages contain the changed texts, but attachments are absent since they cannot be changed
		handler.handle_attachment(message, info.ID, info.MessageSource, info.Timestamp)
	}
	handler.add_to_cache(message, info.ID, info.MessageSource.Chat, info.MessageSource.Sender, info.Timestamp)
}
