package main

/*
#include "constants.h"
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func GetAnyPollCreationMessage(message *waE2E.Message) *waE2E.PollCreationMessage {
	if message.PollCreationMessageV5 != nil {
		return message.PollCreationMessageV5
	}
	if message.PollCreationMessageV3 != nil {
		return message.PollCreationMessageV3
	}
	if message.PollCreationMessageV2 != nil {
		return message.PollCreationMessageV2
	}
	if message.PollCreationMessage != nil {
		return message.PollCreationMessage
	}
	return nil
}

func (handler *Handler) handle_message(message *waE2E.Message, info types.MessageInfo, evt *events.Message) {
	//handler.log.Infof("message: %#v", message)
	if message.SenderKeyDistributionMessage != nil {
		// Apparently, a SenderKeyDistributionMessage can share the a message ID with a conversation message which is to arrive later
		// I do not need this message type in the front-end, so I rather drop it
		// This should be safe (as in "will not inadvertedly drop message with actual payload") since all …Messsage fields seem to be mutually exclusive
		handler.log.Infof("Ignoring SenderKeyDistributionMessage.")
		return
	}
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
				etmText := *etm.Text
				for _, mentioned := range etm.GetContextInfo().GetMentionedJID() {
					mentionedJID, err := types.ParseJID(mentioned)
					if err == nil {
						alias := purple_get_alias(handler.account, handler.lidToPn(mentionedJID, "resolving mention").ToNonAD().String())
						etmText = strings.ReplaceAll(etmText, mentionedJID.User, alias)
					}
				}
				text += etmText
			}
		}

	}
	{
		rm := message.GetReactionMessage()
		if rm != nil && rm.Text != nil && rm.Key != nil && rm.Key.ID != nil {
			quote := fmt.Sprintf("unknown message with ID %s", rm.Key.GetID())
			cached_message := handler.lookup_cached_message_by_id(rm.Key.GetID())
			if cached_message != nil {
				//handler.log.Infof("Lookup yielded message: %#v", &cached_message.Message)
				text := cached_message.Message.GetConversation()
				if cached_message.Message.ExtendedTextMessage != nil {
					text = cached_message.Message.ExtendedTextMessage.GetText()
				}
				if text != "" {
					ellipsis := ""
					if len(text) > 50 {
						ellipsis = "…" // add elipis to indicate message body truncation
					}
					quote = fmt.Sprintf("message „%.50s%s“", text, ellipsis)
				} else {
					message_type := "message of unknown type"
					if cached_message.Message.ImageMessage != nil {
						message_type = "image"
					}
					if cached_message.Message.VideoMessage != nil {
						message_type = "video"
					}
					if cached_message.Message.PtvMessage != nil {
						message_type = "voice message"
					}
					if cached_message.Message.AudioMessage != nil {
						message_type = "audio message"
					}
					if cached_message.Message.StickerMessage != nil {
						message_type = "sticker"
					}
					if cached_message.Message.DocumentMessage != nil {
						message_type = "document"
					}
					quote = fmt.Sprintf("%s from %s", message_type, cached_message.Timestamp.Format(time.RFC822))
				}
			}
			if *rm.Text == "" {
				text += fmt.Sprintf("removed their reaction to %s.", quote)
			} else {
				text += fmt.Sprintf("reacted with %s to %s.", *rm.Text, quote)
			}
		}
	}
	{
		pcm := GetAnyPollCreationMessage(message)
		if pcm != nil {
			//handler.log.Infof("message poll creation: %#v", pcm)
			text = fmt.Sprintf("[POLL] %s\n", pcm.GetName())
			for i, option := range pcm.GetOptions() {
				if option.OptionHash == nil {
					// taken from whatsmeow.HashPollOptions()
					hash := fmt.Sprintf("%X", sha256.Sum256([]byte(option.GetOptionName())))
					option.OptionHash = &hash
				}
				text += fmt.Sprintf("%d: %s\n", i+1, option.GetOptionName())
				//handler.log.Infof("message poll creation option #%d: %#v", i, option)
			}
			selectable_options := pcm.GetSelectableOptionsCount()
			switch selectable_options {
			case 0:
				text += "One may chose multiple answers."
			case 1:
				text += "One may chose one answer."
			default:
				text += fmt.Sprintf("One may chose up to %d answers.", selectable_options)
			}
		}
	}
	{
		pum := message.GetPollUpdateMessage()
		if pum != nil {
			cached_message := handler.lookup_cached_message_by_id(pum.GetPollCreationMessageKey().GetID())
			if cached_message == nil {
				text = "voted in a poll, but this plug-in failed to keep track of the poll."
			} else {
				decrypted, err := handler.client.DecryptPollVote(context.TODO(), evt)
				if err != nil {
					handler.log.Warnf("Failed to decrypt poll vote: %v", err)
				} else {
					pcm := GetAnyPollCreationMessage(&cached_message.Message)
					text = fmt.Sprintf("voted in poll „%s“ for", pcm.GetName())
					if len(decrypted.SelectedOptions) == 0 {
						text += " nothing (removed vote)"
					} else {
						for index, option_hash := range decrypted.SelectedOptions {
							hash := fmt.Sprintf("%X", option_hash)
							var option_name *string = nil
							for _, option := range pcm.Options {
								if option.GetOptionHash() == hash {
									option_name = option.OptionName
								}
							}
							if option_name == nil {
								handler.log.Warnf("Failed look-up poll vote option %s in %#v", hash, &cached_message.Message)
								text += " an unknown option"
							} else {
								separator := ""
								if len(decrypted.SelectedOptions) > 1 {
									if index > 0 {
										separator = ","
									}
									if index == len(decrypted.SelectedOptions)-1 {
										separator = " and"
									}
								}
								text += fmt.Sprintf("%s „%s“", separator, *option_name)
							}
						}
					}
					text += "."
				}
			}
		}
	}
	if text != "" {
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
