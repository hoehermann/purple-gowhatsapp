package main

import (
	"fmt"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"net/http"
)

/*
 * Holds all data for one connection.
 */
type Handler struct {
	client          *whatsmeow.Client
	account         *PurpleAccount
	log             waLog.Logger
	pictureRequests chan ProfilePictureRequest
	httpClient      *http.Client
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
			log.Infof("AppStateSyncComplete")
			err := handler.send_presence(types.PresenceAvailable)
			if err == nil {
				purple_connected(handler.account)
				// connected – start downloading profile pictures now.
				go handler.profile_picture_downloader()
			}
		}
	case *events.PushNameSetting:
		if len(cli.Store.PushName) == 0 {
			return
		}
		// Send presence "available" when the pushname is changed remotely.
		// This makes sure that outgoing messages always have the right pushname.
		handler.send_presence(types.PresenceAvailable)
	case *events.Connected:
		err := handler.send_presence(types.PresenceAvailable)
		if err == nil {
			purple_connected(handler.account)
			// connected – start downloading profile pictures now.
			go handler.profile_picture_downloader()
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
		handler.handle_presence(evt)
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
		handler.handle_chat_presence(evt)
	case *events.AppState:
		log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
	case *events.LoggedOut:
		purple_disconnected(handler.account)
	case *events.QR:
		// this is handled explicitly in connect()
		// though maybe I should move it here for consistency
	case *events.PairSuccess:
		log.Infof("PairSuccess: %#v", evt)
		log.Infof("client.Store: %#v", cli.Store)
		if cli.Store.ID == nil {
			purple_error(handler.account, fmt.Sprintf("Pairing succeded, but device ID is missing."))
		} else {
			credentials := set_credentials(handler.account, *cli.Store.ID, cli.Store.RegistrationID)
			purple_error(handler.account, fmt.Sprintf("Pairing succeeded. Your password is %s. Please reconnect.", credentials))
			handler.prune_devices(*cli.Store.ID)
		}
	case *events.PushName:
		// other device changed our friendly name
		// setting is regarded by whatsmeow internally
		// no need to forward to purple
	default:
		log.Warnf("Event type not handled: %#v", rawEvt)
	}
}
