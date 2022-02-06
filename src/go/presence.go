package main

import (
	"fmt"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"time"
)

func (handler *Handler) send_presence(presence_str string) {
	presenceMap := map[string]types.Presence{
		"available":   types.PresenceAvailable,
		"unavailable": types.PresenceUnavailable,
	}
	presence, ok := presenceMap[presence_str]
	if ok {
		err := handler.client.SendPresence(presence)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("Failed to send presence: %v", err), ERROR_FATAL)
		} else {
			handler.log.Infof("Set presence to %v", presence)
		}
	} else {
		purple_error(handler.account, fmt.Sprintf("Unknown presence %s (this is a bug).", presence_str), ERROR_FATAL)
	}
}

func (handler *Handler) handle_chat_presence(evt *events.ChatPresence) {
	who := evt.MessageSource.Chat.ToNonAD().String()
	switch evt.State {
	case types.ChatPresenceComposing:
		purple_composing(handler.account, who)
	case types.ChatPresencePaused:
		purple_paused(handler.account, who)
	default:
		handler.log.Warnf("ChatPresence not handled: %v", evt.State)
	}
}

func (handler *Handler) handle_presence(evt *events.Presence) {
	if evt.Unavailable {
		purple_update_presence(handler.account, evt.From.ToNonAD().String(), false, evt.LastSeen)
	} else {
		purple_update_presence(handler.account, evt.From.ToNonAD().String(), true, time.Time{})
	}
}

func (handler *Handler) subscribe_presence(who string) {
	jid, err := parseJID(who)
	if err != nil {
		handler.log.Warnf("%s is not a valid JID: %#v", who, err)
		return
	}
	err = handler.client.SubscribePresence(jid)
	if err != nil {
		handler.log.Warnf("Unable to subscribe for presence updates of %s.", jid.String())
	}
}
