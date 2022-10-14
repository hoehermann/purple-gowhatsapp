package main

import (
	"fmt"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"net/http"
	"time"
)

type CachedMessage struct {
	id        types.MessageID
	text      string
	timestamp time.Time
}

/*
 * Holds all data for one connection.
 */
type Handler struct {
	account          *PurpleAccount
	log              waLog.Logger
	container        *sqlstore.Container
	client           *whatsmeow.Client
	deferredReceipts map[types.JID]map[types.JID][]types.MessageID // holds ID and sender of a received message so the receipt can be sent later.
	cachedMessages   []CachedMessage                               // holds texts of some message
	pictureRequests  chan ProfilePictureRequest
	httpClient       *http.Client // for executing picture requests
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
		// this happens after initial logon via QR code (after Connected, but before HistorySync event)
		if evt.Name == appstate.WAPatchCriticalBlock {
			log.Infof("AppStateSyncComplete and WAPatchCriticalBlock")
			handler.handle_connected()
		}
	case *events.PushNameSetting:
		log.Infof("%#v", evt)
		// Send presence when the pushname is changed remotely.
		// This makes sure that outgoing messages always have the right pushname.
		// This is making a round-trip through purple so user can decide to
		// be "away" instead of "online"
		handler.handle_connected()
	case *events.PushName:
		log.Infof("%#v", evt)
		// other device changed our friendly name
		// setting is regarded by whatsmeow internally
		// no need to forward to purple
		// TODO: find out how this is related to the PushNameSetting event
	case *events.Connected:
		// connected – start downloading profile pictures now.
		go handler.profile_picture_downloader()
		handler.handle_connected()
	case *events.StreamReplaced:
		// TODO: find out when exactly this happens and how to handle it (fatal or transient error)
		// working theory: when more than four devices are connected, WhatsApp servers drop the oldest connection
		// NOTE: evt contains no data
		purple_error(handler.account, "Connection stream has been replaced. Reconnecting...", ERROR_TRANSIENT)
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
		purple_error(handler.account, "Logged out. Please link again.", ERROR_FATAL)
	case *events.QR:
		// this is handled explicitly in connect()
		// though maybe I should move it here for consistency
	case *events.PairSuccess:
		log.Infof("PairSuccess: %#v", evt)
		log.Infof("client.Store: %#v", cli.Store)
		if cli.Store.ID == nil {
			purple_error(handler.account, fmt.Sprintf("Pairing succeded, but device ID is missing."), ERROR_FATAL)
		} else {
			set_credentials(handler.account, *cli.Store.ID, cli.Store.RegistrationID)
			purple_pairing_succeeded(handler.account)
			handler.prune_devices(*cli.Store.ID)
		}
	//case *events.JoinedGroup:
	// TODO
	// received when being added to a group directly
	// &events.JoinedGroup{Reason:"", GroupInfo:types.GroupInfo{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"g.us", AD:false}, OwnerJID:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}, GroupName:types.GroupName{Name:"Testgruppe", NameSetAt:time.Date(2020, time.July, 18, 22, 14, 30, 0, time.Local), NameSetBy:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}}, GroupTopic:types.GroupTopic{Topic:"", TopicID:"", TopicSetAt:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), TopicSetBy:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}}, GroupLocked:types.GroupLocked{IsLocked:false}, GroupAnnounce:types.GroupAnnounce{IsAnnounce:false, AnnounceVersionID:"REDACTED"}, GroupEphemeral:types.GroupEphemeral{IsEphemeral:false, DisappearingTimer:0x0}, GroupCreated:time.Date(2020, time.July, 18, 22, 14, 30, 0, time.Local), ParticipantVersionID:"REDACTED", Participants:[]types.GroupParticipant{types.GroupParticipant{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"s.whatsapp.net", AD:false}, IsAdmin:false, IsSuperAdmin:false}, types.GroupParticipant{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"s.whatsapp.net", AD:false}, IsAdmin:true, IsSuperAdmin:false}}}}
	case *events.OfflineSyncCompleted:
	// TODO
	// no idea what this does
	// &events.OfflineSyncCompleted{Count:0}
	default:
		log.Warnf("Event type not handled: %#v", rawEvt)
	}
}
