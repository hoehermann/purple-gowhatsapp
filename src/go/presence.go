package main

import (
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"time"
)

func (handler *Handler) send_presence(presence types.Presence) error {
	err := handler.client.SendPresence(presence)
	if err != nil {
		handler.log.Warnf("Failed to send presence: %v", err)
	} else {
		handler.log.Infof("Set presence to %v", presence)
	}
	return err
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
		if evt.LastSeen.IsZero() {
			handler.log.Infof("%s is now offline", evt.From)
		} else {
			handler.log.Infof("%s is now offline (last seen: %s)", evt.From, evt.LastSeen)
		}
		purple_update_presence(handler.account, evt.From.ToNonAD().String(), false, evt.LastSeen)
	} else {
		handler.log.Infof("%s is now online", evt.From)
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
	} else {
		handler.log.Warnf("Subscribed for presence updates of %s.", jid.String())
	}
}
