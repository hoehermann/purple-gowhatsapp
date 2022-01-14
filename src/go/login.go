package main

/*
#include "constants.h"
*/
import "C"

import (
	"context"
	"fmt"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"strconv"
	"strings"
)

/*
 * This is the go part of purple's login() function.
 */
func login(account *PurpleAccount, credentials string) {
	log := PurpleLogger("Handler")
	_, ok := handlers[account]
	if ok {
		purple_error(account, "This connection already exists.")
		return
	}
	// TODO: protect against concurrent invocation

	var device *store.Device = nil

	// find device (and session) information in database
	// expects user-supplied credentials to be in the form "deviceJid|registrationId".
	// also see set_credentials
	registrationId := uint32(0)
	creds := strings.Split(credentials, "|")
	if len(creds) == 2 {
		deviceJid, err := parseJID(creds[0])
		if err != nil {
			purple_error(account, fmt.Sprintf("Supplied device ID %s is not valid: %#v", err))
			return
		}
		rId, err := strconv.ParseUint(creds[1], 16, 32)
		if err != nil {
			purple_error(account, fmt.Sprintf("Unable to parse registration ID: ", err))
			return
		}
		registrationId = uint32(rId)
		// now query database for device information
		device, err = container.GetDevice(deviceJid)
		if err != nil {
			// this is in case of database errors, presumably
			purple_error(account, fmt.Sprintf("Unable to read device from database: %v", err))
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
		purple_error(account, fmt.Sprintf("Incorrect password."))
		return
	}

	handler := Handler{
		account:         account,
		client:          whatsmeow.NewClient(device, PurpleLogger("Client")),
		log:             log,
		pictureRequests: make(chan ProfilePictureRequest),
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
	devices, err := container.GetAllDevices()
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
			purple_error(handler.account, fmt.Sprintf("%#v", err))
		}
		for evt := range qrChan {
			if !client.IsConnected() {
				clientLog.Infof("Got QR code for disconnected client. Login cancelled.")
				break
			}
			if evt.Event == "code" {
				// Render the QR code here
				size := purple_get_int(handler.account, C.GOWHATSAPP_QRCODE_SIZE_OPTION, 256)
				if size > 0 {
					png, err := qrcode.Encode(evt.Code, qrcode.Medium, size)
					if err != nil {
						purple_error(handler.account, fmt.Sprintf("%#v", err))
					} else {
						purple_display_qrcode(handler.account, evt.Code, png, "")
					}
				} else {
					var b strings.Builder
					fmt.Fprintf(&b, "Scan this code to log in:\n%s\n", evt.Code)
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, &b)
					purple_display_qrcode(handler.account, evt.Code, nil, b.String())
				}
			} else {
				clientLog.Infof("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := client.Connect()
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err))
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
