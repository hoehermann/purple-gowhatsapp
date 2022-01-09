package main

import (
	"fmt"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"time"
)

/*
 * Holds all data for one connection.
 */
type Handler struct {
	client  *whatsmeow.Client
	account *PurpleAccount
	log     waLog.Logger
}

/*
 * This plug-in can handle multiple connections (identified by user-supplied name).
 */
var handlers = make(map[*PurpleAccount]*Handler)

/*
 * Handle incoming events.
 *
 * Largely based on https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go.
 */
func (handler *Handler) eventHandler(rawEvt interface{}) {
	log := handler.log
	cli := handler.client
	switch evt := rawEvt.(type) {
	case *events.AppStateSyncComplete:
		// this happens after initial logon via QR code (before HistorySync)
		if len(cli.Store.PushName) > 0 && evt.Name == appstate.WAPatchCriticalBlock {
			err := cli.SendPresence(types.PresenceAvailable)
			if err != nil {
				log.Warnf("AppStateSyncComplete: Failed to send available presence: %v", err)
			} else {
				log.Infof("AppStateSyncComplete: Marked self as available")
				purple_connected(handler.account)
			}
		}
	case *events.Connected, *events.PushNameSetting:
		if len(cli.Store.PushName) == 0 {
			return
		}
		// Send presence available when connecting and when the pushname is changed.
		// This makes sure that outgoing messages always have the right pushname.
		err := cli.SendPresence(types.PresenceAvailable)
		if err != nil {
			log.Warnf("Failed to send available presence: %v", err)
		} else {
			log.Infof("Marked self as available")
			purple_connected(handler.account)
		}
	case *events.StreamReplaced:
		// TODO: test this
		purple_error(handler.account, fmt.Sprintf("StreamReplaced: %+v", evt))
	case *events.Message:
		handler.handle_message(evt)
	case *events.Receipt:
		if evt.Type == events.ReceiptTypeRead || evt.Type == events.ReceiptTypeReadSelf {
			log.Infof("%v was read by %s at %s", evt.MessageIDs, evt.SourceString(), evt.Timestamp)
		} else if evt.Type == events.ReceiptTypeDelivered {
			log.Infof("%s was delivered to %s at %s", evt.MessageIDs[0], evt.SourceString(), evt.Timestamp)
		}
	case *events.Presence:
		// TODO: find a way to test this
		if evt.Unavailable {
			if evt.LastSeen.IsZero() {
				log.Infof("%s is now offline", evt.From)
			} else {
				log.Infof("%s is now offline (last seen: %s)", evt.From, evt.LastSeen)
			}
			purple_update_presence(handler.account, evt.From.ToNonAD().String(), false, evt.LastSeen)
		} else {
			log.Infof("%s is now online", evt.From)
			purple_update_presence(handler.account, evt.From.ToNonAD().String(), true, time.Time{})
		}
	case *events.HistorySync:
		// this happens after initial logon via QR code (after AppStateSyncComplete)
		log.Infof("history sync: %#v", evt.Data)
		pushnames := evt.Data.GetPushnames()
		for _, p := range pushnames {
			if p.Id != nil && p.Pushname != nil {
				purple_update_name(handler.account, *p.Id, *p.Pushname)
			}
		}
		//conversations := evt.Data.GetConversations() // TODO: enable this if user wants it
	case *events.ChatPresence:
		who := evt.MessageSource.Chat.ToNonAD().String()
		switch evt.State {
		case types.ChatPresenceComposing:
			purple_composing(handler.account, who)
		case types.ChatPresencePaused:
			purple_paused(handler.account, who)
		default:
			log.Warnf("ChatPresence not handled: %v", evt.State)
		}
	case *events.AppState:
		log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
	case *events.LoggedOut:
		purple_disconnected(handler.account)
	case *events.QR:
		// this is handled explicitly in connect()
		// though maybe I should move it here for consistency
	case *events.PairSuccess:
		// jid := evt.ID.ToNonAD().String()
		// TODO: check the registered jid against the user-suppled jid
	case *events.PushName:
		// other device changed our friendly name
		// setting is regarded by whatsmeow internally
		// no need to forward to purple
	default:
		log.Warnf("Event type not handled: %#v", rawEvt)
	}
}
