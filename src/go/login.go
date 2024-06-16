package main

/*
#include "../c/constants.h"
*/
import "C"

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

/*
 * This is the go part of purple's login() function.
 */
func login(account *PurpleAccount, purple_user_dir string, username string, credentials string, proxy_address string) {
	log := PurpleLogger(account, "Handler")
	_, ok := handlers[account]
	if ok {
		purple_error(account, "This connection already exists.", ERROR_FATAL)
		return
	}

	// try to protect against concurrent connections
	// this may lead to problems on multi-user systems since the purple-supplied username not necessarily denotes the actual user JID
	for _, handler := range handlers {
		if handler.username == username {
			purple_error(account, fmt.Sprintf("A connection to this username %s already exists. Please fix your setup.", username), ERROR_FATAL)
			return
		}
	}

	// establish connection to database
	dbLog := PurpleLogger(account, "Database")
	address := purple_get_string(account, C.GOWHATSAPP_DATABASE_ADDRESS_OPTION, C.GOWHATSAPP_DATABASE_ADDRESS_DEFAULT)
	address = strings.Replace(address, "$purple_user_dir", purple_user_dir, -1)
	address = strings.Replace(address, "$username", username, -1)
	dialect := "sqlite3" // see https://github.com/mattn/go-sqlite3/blob/671e666/_example/simple/simple.go#L14
	max_open_conns := 1
	if strings.HasPrefix(address, "postgres:") {
		dialect = "postgres"
		max_open_conns = 0
		address = strings.Replace(address, "postgres:", "", -1)
		// } else if strings.HasPrefix(address, "mysql:") {
		// dialect = "mysql"
		// max_open_conns = 0
		// address = strings.Replace(address, "mysql:", "", -1)
		// disabled until https://github.com/tulir/whatsmeow/pull/48 has been merged
	} else {
		// nothing else, see https://github.com/tulir/whatsmeow/blob/b078a9e/store/sqlstore/container.go#L34
		// and https://github.com/tulir/whatsmeow/blob/4ea4925/mdtest/main.go#L44
	}

	dbLog.Infof("%s connecting to %s", dialect, address)
	db, err := sql.Open(dialect, address)
	if max_open_conns > 0 {
		db.SetMaxOpenConns(max_open_conns)
	}
	if err != nil {
		purple_error(account, fmt.Sprintf("Database driver %s is unable to establish connection to %s due to %v.", dialect, address, err), ERROR_FATAL)
		return
	}
	container := sqlstore.NewWithDB(db, dialect, dbLog)
	err = container.Upgrade()
	if err != nil {
		purple_error(account, fmt.Sprintf("Failed to upgrade database: %v", err), ERROR_FATAL)
		return
	}

	// set our name (displayed in "linked devices")
	store.DeviceProps.Os = proto.String("purple-whatsmeow")

	// limit fetching history since we cannot even parse it
	store.DeviceProps.HistorySyncConfig = &waCompanionReg.DeviceProps_HistorySyncConfig{
		FullSyncDaysLimit:   proto.Uint32(1),
		FullSyncSizeMbLimit: proto.Uint32(1),
		StorageQuotaMb:      proto.Uint32(1),
	}

	// find device (and session) information in database
	// expects user-supplied credentials to be in the form "deviceJid|registrationId".
	// also see set_credentials
	var device *store.Device = nil
	registrationId := uint32(0)
	creds := strings.Split(credentials, "|")
	if len(creds) == 2 {
		deviceJid, err := parseJID(creds[0])
		if err != nil {
			purple_error(account, fmt.Sprintf("Supplied device ID %v is not valid: %#v", deviceJid, err), ERROR_FATAL)
			return
		}
		rId, err := strconv.ParseUint(creds[1], 16, 32)
		if err != nil {
			purple_error(account, fmt.Sprintf("Unable to parse registration ID: %#v", err), ERROR_FATAL)
			return
		}
		registrationId = uint32(rId)
		// now query database for device information
		device, err = container.GetDevice(deviceJid)
		if err != nil {
			// this is in case of database errors, presumably
			purple_error(account, fmt.Sprintf("Unable to read device from database: %#v", err), ERROR_FATAL)
			return
		}
	}

	if device == nil {
		// device == nil happens in case the database contained no appropriate device
		device = container.NewDevice()
		// device.RegistrationID has been generated. sync it and continue (see next if below)
		registrationId = device.RegistrationID
	} else if device.ID.ToNonAD().String() != username {
		purple_error(account, fmt.Sprintf("Your username '%s' does not match the main device's ID '%s'. Please adjust your username.", username, device.ID.ToNonAD().String()), ERROR_FATAL)
	}

	// check user-supplied registration id against the one stored in the device
	// this is necessary for multi-user set-ups like spectrum or bitlbee
	// where we cannot universally trust all local users.
	// we must employ a mechanism that checks against some secret
	// so it is not sufficient to know a person's device ID to hijack their account
	// there is nothing special about the RegistrationID. any of the fields could be used.
	if device.RegistrationID != registrationId {
		purple_error(account, "Incorrect credentials.", ERROR_FATAL)
		return
	}

	// try to protect against concurrent connections
	// look through all currently active connections
	// abort if there already is a connection with this JID
	for _, handler := range handlers {
		if handler.client.Store != nil && handler.client.Store.ID != nil && device.ID != nil && handler.client.Store.ID.ToNonAD() == device.ID.ToNonAD() {
			purple_error(account, fmt.Sprintf("A connection to this number %s already exists. Please fix your setup.", device.ID.String()), ERROR_FATAL)
			return
		}
	}

	handler := Handler{
		account:          account,
		username:         username,
		container:        container,
		log:              log,
		client:           whatsmeow.NewClient(device, PurpleLogger(account, "Client")),
		deferredReceipts: make(map[types.JID]map[types.JID][]types.MessageID),
		pictureRequests:  make(chan ProfilePictureRequest),
	}
	handlers[account] = &handler
	handler.client.AddEventHandler(handler.eventHandler)

	if proxy_address != "" {
		handler.client.SetProxyAddress(proxy_address)
	}
	err = handler.client.Connect()
	if err != nil {
		purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_TRANSIENT)
		return
	}
}

/*
 * Generates an pairing 8-character pairing code.
 */
func (handler *Handler) generate_pairing_code() (string, error) {
	own_jid, err := parseJID(handler.username)
	if err != nil {
		return "", fmt.Errorf("„%s“ is not a valid WhatsApp JID.", handler.username)
	}
	phone := own_jid.ToNonAD().User
	showPushNotification := true
	// cannot use store.DeviceProps.Os here, since "only common browsers/OSes are allowed",
	// see https://pkg.go.dev/go.mau.fi/whatsmeow#Client.PairPhone
	// Firefox on Linux chosen arbitrarily
	clientDisplayName := "Firefox (Linux)"
	clientType := whatsmeow.PairClientFirefox
	pairing_code, err := handler.client.PairPhone(phone, showPushNotification, clientType, clientDisplayName)
	return pairing_code, err
}

/*
 * After calling client.Connect() with a pristine device ID, WhatsApp servers
 * send a list of codes which can be turned into QR codes for scanning with the offical app.
 */
func (handler *Handler) handle_qrcode(qrcodes []string) {
	var stringBuilder strings.Builder
	pairing_code, err := handler.generate_pairing_code()
	if err != nil {
		purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
	}
	qrcode_data := qrcodes[0] // use only first code for now
	// TODO: emit events to destroy and update the code in the ui
	fmt.Fprintf(&stringBuilder, "Enter pairing code %s or scan this code to log in:\n%s\n", pairing_code, qrcode_data)
	qrterminal.GenerateHalfBlock(qrcode_data, qrterminal.L, &stringBuilder)
	size := purple_get_int(handler.account, C.GOWHATSAPP_QRCODE_SIZE_OPTION, 256)
	var png []byte
	if size > 0 {
		png, err = qrcode.Encode(qrcode_data, qrcode.Medium, size)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
		}
	}
	purple_display_qrcode(handler.account, stringBuilder.String(), qrcode_data, png)
}

/*
 * Store the credentials (deviceJID and a "password").
 * The credentials are munged into a single string for bitlbee compatibility.
 */
func set_credentials(account *PurpleAccount, deviceJid types.JID, registrationId uint32) string {
	rId := fmt.Sprintf("%x", registrationId)
	dJ := deviceJid.String()
	creds := fmt.Sprintf("%s|%s", dJ, rId)
	purple_set_credentials(account, creds)
	return creds
}

func (handler *Handler) prune_devices(deviceJid types.JID) {
	if handler.container == nil {
		purple_error(handler.account, "prune_devices called without a database connection", ERROR_FATAL)
		return
	}
	devices, err := handler.container.GetAllDevices()
	if err == nil {
		for _, device := range devices {
			if device.ID == nil {
				handler.log.Infof("Deleting bogous device %s from database...", device.ID.String())
				device.Delete() // ignores errors
			} else {
				obsolete := device.ID.ToNonAD() == deviceJid.ToNonAD() && *device.ID != deviceJid
				if obsolete {
					handler.log.Infof("Deleting obsolete device %s from database...", device.ID.String())
					device.Delete() // ignores errors
				}
			}
		}
	}
}

/*
 * This is the go part of purple's close() function.
 */
func close(account *PurpleAccount) {
	handler, ok := handlers[account]
	if ok {
		// tell the background downloader to terminate
		select {
		case handler.pictureRequests <- ProfilePictureRequest{}:
			// termination request sent
			// nothing to do here
		default:
			// termination request not sent
			// ignore silently and continue
		}
		handler.client.Disconnect()
		delete(handlers, account)
	}
}
