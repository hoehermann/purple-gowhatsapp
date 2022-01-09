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
	"strings"
)

/*
 * This is the go part of purple's login() function.
 */
func login(account *PurpleAccount) {
	log := PurpleLogger("Handler")
	_, ok := handlers[account]
	if ok {
		purple_error(account, "This connection already exists.")
	}
	// TODO: protect against concurrent invocation
	deviceStore, err := container.GetFirstDevice() // TODO: find out how to use a device jid and use .GetDevice(jid)
	if err != nil {
		purple_error(account, fmt.Sprintf("%#v", err))
	} else {
		handler := Handler{
			account: account,
			client:  whatsmeow.NewClient(deviceStore, PurpleLogger("Client")),
			log:     log,
		}
		handlers[account] = &handler
		handler.client.AddEventHandler(handler.eventHandler)
		go handler.connect()
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
