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
	"strconv"
	"strings"
)

/*
 * This is the go part of purple's login() function.
 */
func login(account *PurpleAccount, username string, password string) {
	log := PurpleLogger("Handler")
	_, ok := handlers[account]
	if ok {
		purple_error(account, "This connection already exists.")
		return
	}
	// TODO: protect against concurrent invocation

	var device *store.Device = nil
	deviceJid, err := parseJID(username)
	if err != nil {
		purple_error(account, fmt.Sprintf("Supplied username %s is not a valid device ID: %#v", err))
		return
	} else {
		device, err = container.GetDevice(deviceJid)
	}
	if err != nil {
		// this is in case of database errors, presumably
		purple_error(account, fmt.Sprintf("Unable to read device from database: %v", err))
		return
	}

	rId, err := strconv.ParseUint(password, 16, 32)
	registrationId := uint32(rId)
	if err != nil {
		purple_error(account, fmt.Sprintf("Unable to parse password into numerical registration id: ", err))
		return
	}

	if device == nil {
		// device == nil happens in case the database contained no appropriate device
		device = container.NewDevice()
		password = set_credentials(account, username, device.RegistrationID)
		registrationId = device.RegistrationID
	}

	// check user-supplied registration id against the one stored in the device
	// this is necessary for multi-user set-ups like spectrum or bitlbee
	// where we cannot universally trust all local users
	if device.RegistrationID != registrationId {
		purple_error(account, fmt.Sprintf("Incorrect registration ID. Enter a different username to create a new pairing."))
		return
	}

	handler := Handler{
		account: account,
		client:  whatsmeow.NewClient(device, PurpleLogger("Client")),
		log:     log,
	}
	handlers[account] = &handler
	handler.client.AddEventHandler(handler.eventHandler)
	go handler.connect()
}

func set_credentials(account *PurpleAccount, username string, registrationId uint32) string {
	rId := fmt.Sprintf("%x", registrationId)
	purple_set_credentials(account, username, rId)
	return rId
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
		handler.client.Disconnect()
		delete(handlers, account)
	}
}
