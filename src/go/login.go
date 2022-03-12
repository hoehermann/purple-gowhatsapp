package main

/*
#include "../c/constants.h"
*/
import "C"

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"strconv"
	"strings"
)

/*
 * This is the go part of purple's login() function.
 */
func login(account *PurpleAccount, purple_user_dir string, username string, credentials string) {
	log := PurpleLogger(account, "Handler")
	_, ok := handlers[account]
	if ok {
		purple_error(account, "This connection already exists.", ERROR_FATAL)
		return
	}
	// TODO: protect against concurrent invocation

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
	} else if strings.HasPrefix(address, "mysql:") {
		dialect = "mysql"
		max_open_conns = 0
		address = strings.Replace(address, "mysql:", "", -1)
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
		purple_error(account, fmt.Sprintf("whatsmeow database driver is unable to establish connection to %s due to %v.", address, err), ERROR_FATAL)
		return
	}
	container := sqlstore.NewWithDB(db, dialect, dbLog)

	// find device (and session) information in database
	// expects user-supplied credentials to be in the form "deviceJid|registrationId".
	// also see set_credentials
	var device *store.Device = nil
	registrationId := uint32(0)
	creds := strings.Split(credentials, "|")
	if len(creds) == 2 {
		deviceJid, err := parseJID(creds[0])
		if err != nil {
			purple_error(account, fmt.Sprintf("Supplied device ID %s is not valid: %#v", err), ERROR_FATAL)
			return
		}
		rId, err := strconv.ParseUint(creds[1], 16, 32)
		if err != nil {
			purple_error(account, fmt.Sprintf("Unable to parse registration ID: ", err), ERROR_FATAL)
			return
		}
		registrationId = uint32(rId)
		// now query database for device information
		device, err = container.GetDevice(deviceJid)
		if err != nil {
			// this is in case of database errors, presumably
			purple_error(account, fmt.Sprintf("Unable to read device from database: %v", err), ERROR_FATAL)
			return
		}
	}

	if device == nil {
		// device == nil happens in case the database contained no appropriate device
		device = container.NewDevice()
		// device.RegistrationID has been generated. sync it and continue (see next if below)
		registrationId = device.RegistrationID
	}

	// check user-supplied registration id against the one stored in the device
	// this is necessary for multi-user set-ups like spectrum or bitlbee
	// where we cannot universally trust all local users.
	// we must employ a mechanism that checks against some secret
	// so it is not sufficient to know a person's device ID to hijack their account
	// there is nothing special about the RegistrationID. any of the fields could be used.
	if device.RegistrationID != registrationId {
		purple_error(account, fmt.Sprintf("Incorrect credentials."), ERROR_FATAL)
		return
	}

	handler := Handler{
		account:          account,
		container:        container,
		log:              log,
		client:           whatsmeow.NewClient(device, PurpleLogger(account, "Client")),
		deferredReceipts: make(map[types.JID]map[types.JID][]types.MessageID),
		pictureRequests:  make(chan ProfilePictureRequest),
	}
	handlers[account] = &handler
	handler.client.AddEventHandler(handler.eventHandler)
	go handler.connect()
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
 * Helper function for login procedure.
 * Calls whatsmeow.Client.Connect().
 */
func (handler *Handler) connect() {
	clientLog := handler.log
	client := handler.client
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
		}
		for evt := range qrChan {
			if !client.IsConnected() {
				clientLog.Infof("Got QR code for disconnected client. Login cancelled.")
				break
			}
			if evt.Event == "code" {
				// Render the QR code here
				var png []byte
				var b strings.Builder
				fmt.Fprintf(&b, "Scan this code to log in:\n%s\n", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, &b)
				size := purple_get_int(handler.account, C.GOWHATSAPP_QRCODE_SIZE_OPTION, 256)
				if size > 0 {
					png, err = qrcode.Encode(evt.Code, qrcode.Medium, size)
					if err != nil {
						purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
					}
				}
				purple_display_qrcode(handler.account, b.String(), evt.Code, png)
			} else {
				clientLog.Infof("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := client.Connect()
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_TRANSIENT)
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
