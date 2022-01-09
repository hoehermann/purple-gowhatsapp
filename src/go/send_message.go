package main

import (
	"fmt"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"strings"
)

// from https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func (handler *Handler) parseJID(arg string) (types.JID, bool) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("Invalid JID %s: %v", arg, err))
			return recipient, false
		} else if recipient.User == "" {
			purple_error(handler.account, fmt.Sprintf("Invalid JID %s: no server specified", arg))
			return recipient, false
		}
		return recipient, true
	}
}

func (handler *Handler) send_message(who string, message string, isGroup bool) {
	recipient, ok := handler.parseJID(who) // calls purple_error directly
	if ok {
		msg := &waProto.Message{Conversation: &message}
		ts, err := handler.client.SendMessage(recipient, "", msg)
		if err != nil {
			// TODO: display error in conversation
			handler.log.Errorf("Error sending message: %v", err)
		} else {
			handler.log.Infof("Message sent (server timestamp: %s)", ts)
			// inject message back to self to indicate success
			ownJid := "" // TODO: find out if this messes up group chats
			purple_display_text_message(handler.account, recipient.ToNonAD().String(), isGroup, true, ownJid, nil, ts, message)
		}
	}

}
