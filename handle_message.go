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

func (handler *Handler) handle_message(message *waE2E.Message, id string, source types.MessageSource, name *string, timestamp time.Time, is_historical bool) {
	//handler.log.Infof("message: %#v", message)
	if source.Chat == types.StatusBroadcastJID {
		if purple_get_bool(handler.account, C.GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, false) {
			// some people find status broadcasts annoying
			handler.log.Warnf("Ignoring status broadcast.")
			return
		} else {
			// the protocol implements status broadcasts in the form of a group
			// we just treat those messages as if they were direct messages
			source.Chat = source.Sender
			source.IsGroup = false
		}
	}
	if handler.blocklist != nil {
		// TODO find out whether locally checking the blocklist is actually necessary or if WhatsApp servers do the filtering for us
		for _, blockedJID := range handler.blocklist.JIDs {
			if blockedJID.ToNonAD() == source.Sender.ToNonAD() {
				handler.log.Infof("Ignoring message from %s since they are on the blocklist.", source.Sender.ToNonAD().String())
			}
		}
	}
	text := ""
	{
		if pm := message.GetProtocolMessage(); pm != nil {
			if em := pm.GetEditedMessage(); em != nil {
				message = em
				text = "[EDIT] "
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
			// the look-up currently does not work for outgoing image messages
			// TODO: add/check all kinds of outgoing messages (I do not remember if text-messages are already working)
			for i := range handler.cachedMessages {
				if handler.cachedMessages[i].id == rm.Key.GetID() {
					message := &handler.cachedMessages[i]
					quote = fmt.Sprintf("message \"%.50s\" from %s", message.text, message.timestamp.Format(time.RFC822))
					// TODO: add elipis to indicate message truncation
					break
				}
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
		// note: info.PushName always denotes the sender (not the chat)
		purple_display_text_message(handler.account, source.Chat.ToNonAD().String(), source.IsGroup, false, source.Sender.ToNonAD().String(), name, timestamp, text, &id)
		handler.addToCache(CachedMessage{id: id, text: text, timestamp: timestamp})
		if !source.IsFromMe && !is_historical { // do not send receipt for own messages or historical messages
			handler.mark_read_defer(id, source.Chat, source.Sender)
			handler.mark_read_if_on_receival(source.Chat)
		}
	}
	handler.handle_attachment(message, id, source, timestamp)
}
