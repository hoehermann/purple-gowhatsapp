package main

/*
#include "constants.h"
*/
import "C"

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
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
	username         string
	log              waLog.Logger
	container        *sqlstore.Container
	client           *whatsmeow.Client
	deferredReceipts map[types.JID]map[types.JID][]types.MessageID // holds ID and sender of a received message so the receipt can be sent later.
	cachedMessages   []CachedMessage                               // for looking up reactions and quotes
	pictureRequests  chan ProfilePictureRequest
	httpClient       *http.Client // for executing picture requests
	blocklist        *types.Blocklist
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
		// Using SetPassive is a custom feature requested by https://github.com/theassemblerguy
		if purple_get_bool(handler.account, C.GOWHATSAPP_PASSIVE_OPTION, false) {
			handler.client.SetPassive(context.TODO(), true)
		}
		// connected – start downloading profile pictures now.
		go handler.profile_picture_downloader()
		handler.handle_connected()
		blocklist, err := cli.GetBlocklist(context.TODO())
		if err == nil {
			log.Infof("Blocklist contains %d entries.", len(blocklist.JIDs))
			handler.blocklist = blocklist
		} else {
			log.Warnf("Failed to obtain blocklist due to %#v.", err)
		}
	case *events.Disconnected:
		// TODO: Find out if it would be more sensible to handle this as a non-error disconnect.
		purple_error(handler.account, "Disconnected.", ERROR_TRANSIENT)
	case *events.StreamReplaced:
		// TODO: find out when exactly this happens and how to handle it (fatal or transient error)
		// working theory: when more than four devices are connected, WhatsApp servers drop the oldest connection
		// NOTE: evt contains no data
		purple_error(handler.account, "Connection stream has been replaced. Reconnecting...", ERROR_TRANSIENT)
	case *events.KeepAliveTimeout:
		// ignore the lost connection since whatsmeow is pretty good at re-establishing
		// though trying to send a message will end with a time-out
		// TODO: reflect this stte in the UI, maybe by setting purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTING);
		log.Warnf("KeepAlive timed out. Reconnecting in background...")
	case *events.Message:
		handler.handle_message(evt.Message, evt.Info.ID, evt.Info.MessageSource, &evt.Info.PushName, evt.Info.Timestamp, false)
	case *events.Receipt:
		if evt.Type == types.ReceiptTypeRead || evt.Type == types.ReceiptTypeReadSelf {
			log.Infof("%v was read by %s at %s", evt.MessageIDs, evt.SourceString(), evt.Timestamp)
		} else if evt.Type == types.ReceiptTypeDelivered {
			log.Infof("%s was delivered to %s at %s", evt.MessageIDs[0], evt.SourceString(), evt.Timestamp)
		}
	case *events.Presence:
		handler.handle_presence(evt)
	case *events.ChatPresence:
		handler.handle_chat_presence(evt)
	case *events.Picture:
		handler.request_profile_picture(evt.JID, "", evt.PictureID)
	case *events.HistorySync:
		// this happens after initial logon via QR code (after AppStateSyncComplete)
		if purple_get_bool(handler.account, C.GOWHATSAPP_FETCH_CONTACTS_AFTER_LINKING_OPTION, true) {
			pushnames := evt.Data.GetPushnames()
			for _, p := range pushnames {
				if p.ID != nil && p.Pushname != nil {
					handler.log.Infof("HistorySync Pushname: %#v", p)
					purple_update_name(handler.account, *p.ID, *p.Pushname)
				}
			}
		}
		// TODO: handle historical conversations obtained by evt.Data.GetConversations() utilising client.ParseWebMessage
	case *events.AppState:
		log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
	case *events.LoggedOut:
		purple_error(handler.account, "Logged out. Please link again.", ERROR_FATAL)
	case *events.QR:
		handler.handle_qrcode(evt.Codes)
	case *events.PairSuccess:
		log.Infof("PairSuccess: %#v", evt)
		log.Infof("client.Store: %#v", cli.Store)
		if cli.Store.ID == nil {
			purple_error(handler.account, "Pairing succeded, but device ID is missing.", ERROR_FATAL)
		} else if evt.ID.ToNonAD().String() != handler.username {
			purple_error(handler.account, fmt.Sprintf("Your username '%s' does not match the main device's ID '%s'. Please adjust your username.", handler.username, evt.ID.ToNonAD().String()), ERROR_FATAL)
		} else {
			set_credentials(handler.account, *cli.Store.ID, cli.Store.RegistrationID)
			purple_pairing_succeeded(handler.account)
			handler.prune_devices(*cli.Store.ID)
		}
	case *events.CallOffer:
		bcm := evt.BasicCallMeta
		chat := handler.lidToPn(bcm.From, "handling call offer")
		sender := handler.lidToPn(bcm.CallCreator, "handling call offer")
		text := "This contact is trying to call you, but WhatsApp Web does not support calls."
		purple_display_text_message(handler.account, chat.ToNonAD().String(), false, false, sender.ToNonAD().String(), nil, bcm.Timestamp, text, nil)
	case *events.CallOfferNotice:
		// same as CallOffer, but is a group
		bcm := evt.BasicCallMeta
		chat := handler.lidToPn(bcm.From, "handling call offer notice")
		sender := handler.lidToPn(bcm.CallCreator, "handling call offer notice")
		text := "This contact is trying to make you notice a call, but WhatsApp Web does not support calls."
		purple_display_text_message(handler.account, chat.ToNonAD().String(), true, false, sender.ToNonAD().String(), nil, bcm.Timestamp, text, nil)
	case *events.CallRelayLatency:
		// related to calls. ignore silently.
	case *events.CallTerminate:
		// related to calls. ignore silently.
	//case *events.JoinedGroup:
	// TODO
	// received when being added to a group directly
	// NOTE: Spectrum users are not notified if they have been joined to a group. Their XMPP client always needs to join explicitly.
	// &events.JoinedGroup{Reason:"", GroupInfo:types.GroupInfo{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"g.us", AD:false}, OwnerJID:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}, GroupName:types.GroupName{Name:"Testgruppe", NameSetAt:time.Date(2020, time.July, 18, 22, 14, 30, 0, time.Local), NameSetBy:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}}, GroupTopic:types.GroupTopic{Topic:"", TopicID:"", TopicSetAt:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), TopicSetBy:types.JID{User:"", Agent:0x0, Device:0x0, Server:"", AD:false}}, GroupLocked:types.GroupLocked{IsLocked:false}, GroupAnnounce:types.GroupAnnounce{IsAnnounce:false, AnnounceVersionID:"REDACTED"}, GroupEphemeral:types.GroupEphemeral{IsEphemeral:false, DisappearingTimer:0x0}, GroupCreated:time.Date(2020, time.July, 18, 22, 14, 30, 0, time.Local), ParticipantVersionID:"REDACTED", Participants:[]types.GroupParticipant{types.GroupParticipant{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"s.whatsapp.net", AD:false}, IsAdmin:false, IsSuperAdmin:false}, types.GroupParticipant{JID:types.JID{User:"REDACTED", Agent:0x0, Device:0x0, Server:"s.whatsapp.net", AD:false}, IsAdmin:true, IsSuperAdmin:false}}}}
	case *events.OfflineSyncCompleted:
	// TODO
	// no idea what this does
	// &events.OfflineSyncCompleted{Count:0}
	case *events.Blocklist:
	// TODO update local blocklist
	// &events.Blocklist{Action:"", DHash:"REDACTED", PrevDHash:"REDACTED", Changes:[]events.BlocklistChange{events.BlocklistChange{JID:types.JID{User:"REDACTED", RawAgent:0x0, Device:0x0, Integrator:0x0, Server:"s.whatsapp.net"}, Action:"block"}}}
	case *events.UndecryptableMessage:
		info := evt.Info
		source := info.MessageSource
		text := fmt.Sprintf("sent an undecryptable %s message. Check the message on your main device.", evt.UnavailableType)
		purple_display_text_message(handler.account, source.Chat.ToNonAD().String(), source.IsGroup, false, source.Sender.ToNonAD().String(), &info.PushName, info.Timestamp, text, &info.ID)
	default:
		log.Warnf("Event type not handled: %#v", rawEvt)
	}
}
