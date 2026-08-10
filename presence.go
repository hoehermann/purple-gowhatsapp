package main

/*
#include "constants.h"
*/
import "C"

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

/*
 * Forward whatsmeow connection event to purple.
 * In case the PushName is not set, the connection is not (yet) actually usable.
 */
func (handler *Handler) handle_connected() {
	if len(handler.client.Store.PushName) > 0 {
		purple_connected(handler.account)
	}
}

func (handler *Handler) send_presence(presence_str string) {
	presenceMap := map[string]types.Presence{
		"available":   types.PresenceAvailable,
		"unavailable": types.PresenceUnavailable,
	}
	presence, ok := presenceMap[presence_str]
	if ok {
		err := handler.client.SendPresence(context.TODO(), presence)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("Failed to send presence: %v", err), ERROR_FATAL)
		}
	} else {
		purple_error(handler.account, fmt.Sprintf("Unknown presence %s (this is a bug).", presence_str), ERROR_FATAL)
	}
}

func (handler *Handler) handle_chat_presence(evt *events.ChatPresence) {
	who := handler.lidToPn(evt.MessageSource.Chat, "handling chat (typing) presence").ToNonAD().String()
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
	who := handler.lidToPn(evt.From, "handling presence").ToNonAD().String()
	if evt.Unavailable {
		purple_update_presence(handler.account, who, false, evt.LastSeen)
	} else {
		purple_update_presence(handler.account, who, true, time.Time{})
	}
}

func (handler *Handler) subscribe_presence(who string) {
	jid, err := parseJID(who)
	if err != nil {
		handler.log.Warnf("%s is not a valid JID: %#v", who, err)
		return
	}
	err = handler.client.SubscribePresence(context.TODO(), jid)
	if err != nil {
		handler.log.Warnf("Unable to subscribe for presence updates of %s.", jid.String())
	}
}

/*
 * Informs the other party this user started or stopped composing a message.
 */
func (handler *Handler) send_typing(who string, typing bool) {
	recipient, err := types.ParseJID(who)
	if err != nil {
		handler.log.Warnf("send_typing: %v is not a valid JID: %v", who, err)
		return
	}
	state := types.ChatPresencePaused
	if typing {
		state = types.ChatPresenceComposing
	}
	err = handler.client.SendChatPresence(context.Background(), recipient, state, types.ChatPresenceMediaText)
	if err != nil {
		handler.log.Warnf("send_typing failed: %v", err)
	}
}
