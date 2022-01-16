package main

import (
	"fmt"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"strings"
)

// from https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func parseJID(arg string) (types.JID, error) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), nil
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			return recipient, fmt.Errorf("Invalid JID %s: %v", arg, err)
		} else if recipient.User == "" {
			return recipient, fmt.Errorf("Invalid JID %s: no server specified", arg)
		}
		return recipient, nil
	}
}

func (handler *Handler) send_message(who string, message string, isGroup bool) {
	recipient, err := parseJID(who)
	if err != nil {
		purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
	} else {
		msg := &waProto.Message{Conversation: &message}
		ts, err := handler.client.SendMessage(recipient, "", msg)
		if err != nil {
			// TODO: display error in conversation
			errmsg := fmt.Sprintf("Error sending message: %v", err)
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, errmsg)
		} else {
			//handler.log.Infof("Message sent (server timestamp: %s)", ts)
			// inject message back to self to indicate success
			ownJid := "" // TODO: find out if this messes up group chats
			purple_display_text_message(handler.account, recipient.ToNonAD().String(), isGroup, true, ownJid, nil, ts, message)
		}
		handler.mark_read_if_on_answer(recipient)
	}

}
