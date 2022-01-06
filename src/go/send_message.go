package main

import (
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
			handler.log.Errorf("Invalid JID %s: %v", arg, err)
			return recipient, false
		} else if recipient.User == "" {
			handler.log.Errorf("Invalid JID %s: no server specified", arg)
			return recipient, false
		}
		return recipient, true
	}
}

func (handler *Handler) send_message(username string, who string, message string) {
	recipient, ok := handler.parseJID(who)
	if ok {
		msg := &waProto.Message{Conversation: &message}
		ts, err := handler.client.SendMessage(recipient, "", msg)
		if err != nil {
			handler.log.Errorf("Error sending message: %v", err)
		} else {
			handler.log.Infof("Message sent (server timestamp: %s)", ts)
			purple_display_text_message(handler.username, nil, recipient.ToNonAD().String(), false, true, handler.username, nil, ts, message, nil)
		}
	}

}
