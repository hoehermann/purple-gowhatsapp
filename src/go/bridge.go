/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2022 Hermann Höhne
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

/*
 * Apparently, the main package must be named main, even though this is a library
 */
package main

/*
#include "constants.h"
#include "bridge.h"
*/
import "C"

import "time"

type PurpleAccount = C.PurpleAccount

// TODO: find out how to enable C99's bool type in cgo
func bool_to_Cchar(b bool) C.char {
	if b {
		return C.char(1)
	} else {
		return C.char(0)
	}
}

func Cint_to_bool(i C.int) bool {
	return i != 0
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(account *PurpleAccount, purple_user_dir *C.char, username *C.char, password *C.char) {
	login(account, C.GoString(purple_user_dir), C.GoString(username), C.GoString(password))
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(account *PurpleAccount) {
	close(account)
}

//export gowhatsapp_go_send_message
func gowhatsapp_go_send_message(account *PurpleAccount, who *C.char, message *C.char, is_group C.int) int {
	handler, ok := handlers[account]
	if ok {
		go handler.send_message(C.GoString(who), C.GoString(message), Cint_to_bool(is_group))
		return 0
	}
	return -107 // ENOTCONN, see libpurple/prpl.h
}

//export gowhatsapp_go_send_file
func gowhatsapp_go_send_file(account *PurpleAccount, who *C.char, filename *C.char) int {
	handler, ok := handlers[account]
	if ok {
		return handler.send_file(C.GoString(who), C.GoString(filename))
	}
	return -107 // ENOTCONN, see libpurple/prpl.h
}

//export gowhatsapp_go_mark_read_conversation
func gowhatsapp_go_mark_read_conversation(account *PurpleAccount, who *C.char) {
	handler, ok := handlers[account]
	if ok {
		handler.mark_read_conversation(C.GoString(who))
	} else {
		// no connection, fail silently
	}
}

//export gowhatsapp_go_subscribe_presence
func gowhatsapp_go_subscribe_presence(account *PurpleAccount, who *C.char) {
	handler, ok := handlers[account]
	if ok {
		handler.subscribe_presence(C.GoString(who))
	} else {
		// no connection, fail silently
	}
}

//export gowhatsapp_go_request_profile_picture
func gowhatsapp_go_request_profile_picture(account *PurpleAccount, who *C.char, picture_date *C.char) {
	handler, ok := handlers[account]
	if ok {
		go handler.request_profile_picture(C.GoString(who), C.GoString(picture_date))
	} else {
		// no connection, fail silently
	}
}

/*
 * This will display a QR code via PurpleRequest API.
 */
func purple_display_qrcode(account *PurpleAccount, terminal string, challenge string, png []byte) {
	cmessage := C.struct_gowhatsapp_message{
		account:  account,
		msgtype:  C.char(C.gowhatsapp_message_type_login),
		text:     C.CString(challenge),
		name:     C.CString(terminal),
		blob:     C.CBytes(png),
		blobsize: C.size_t(len(png)),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will close the QR code shown previously.
 */
func purple_pairing_succeeded(account *PurpleAccount) {
	cmessage := C.struct_gowhatsapp_message{
		account: account,
		msgtype: C.char(C.gowhatsapp_message_type_pairing_succeeded),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will inform purple that the account has been connected.
 */
func purple_connected(account *PurpleAccount) {
	cmessage := C.struct_gowhatsapp_message{
		account: account,
		msgtype: C.char(C.gowhatsapp_message_type_connected),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will inform purple that the account has been disconnected.
 */
func purple_disconnected(account *PurpleAccount) {
	cmessage := C.struct_gowhatsapp_message{
		account: account,
		msgtype: C.char(C.gowhatsapp_message_type_disconnected),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will display a text message.
 * Single participants and group chats.
 */
func purple_display_text_message(account *PurpleAccount, remoteJid string, isGroup bool, isFromMe bool, senderJid string, pushName *string, timestamp time.Time, text string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_text),
		remoteJid: C.CString(remoteJid),
		senderJid: C.CString(senderJid),
		timestamp: C.time_t(timestamp.Unix()),
		text:      C.CString(text),
		isGroup:   bool_to_Cchar(isGroup),
		fromMe:    bool_to_Cchar(isFromMe),
	}
	if pushName != nil {
		cmessage.name = C.CString(*pushName)
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will display a system message.
 * For soft errors regarding a specific conversation.
 * Single participants and group chats.
 */
func purple_display_system_message(account *PurpleAccount, remoteJid string, isGroup bool, text string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_system),
		remoteJid: C.CString(remoteJid),
		text:      C.CString(text),
		isGroup:   bool_to_Cchar(isGroup),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will update a contact's name.
 * May add them to the local buddy list.
 * Does work for individuals, not for groups.
 */
func purple_update_name(account *PurpleAccount, remoteJid string, pushName string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_name),
		remoteJid: C.CString(remoteJid),
		name:      C.CString(pushName),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will create a file transfer for receiving an attachment.
 * Works well for participants. A bit wonky for group chats.
 *
 * Please note: Pidgin will ask the user if they wish to receive the file
 * while in fact the file has already been received and they may only chose
 * where to store it.
 */
func purple_handle_attachment(account *PurpleAccount, senderJid string, filename string, data []byte) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_attachment),
		senderJid: C.CString(senderJid),
		name:      C.CString(filename),
		blob:      C.CBytes(data),
		blobsize:  C.size_t(len(data)), // contrary to https://golang.org/pkg/builtin/#len and https://golang.org/ref/spec#Numeric_types, len returns an int of 64 bits on 32 bit Windows machines (see https://github.com/hoehermann/purple-gowhatsapp/issues/1)
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * Forwards a downloaded profile picture to purple.
 */
func purple_set_profile_picture(account *PurpleAccount, who string, data []byte, picture_date string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_profile_picture),
		remoteJid: C.CString(who),
		text:      C.CString(picture_date),
		blob:      C.CBytes(data),
		blobsize:  C.size_t(len(data)),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will inform purple that the remote user started typing.
 */
func purple_composing(account *PurpleAccount, remoteJid string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_typing),
		remoteJid: C.CString(remoteJid),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will inform purple that the remote user stopped typing.
 */
func purple_paused(account *PurpleAccount, remoteJid string) {
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_typing_stopped),
		remoteJid: C.CString(remoteJid),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will inform purple that the remote user's presence (online/offline) changed.
 */
func purple_update_presence(account *PurpleAccount, remoteJid string, online bool, lastSeen time.Time) {
	timestamp := C.time_t(0)
	if !lastSeen.IsZero() {
		timestamp = C.time_t(lastSeen.Unix())
	}
	cmessage := C.struct_gowhatsapp_message{
		account:   account,
		msgtype:   C.char(C.gowhatsapp_message_type_presence),
		remoteJid: C.CString(remoteJid),
		timestamp: timestamp,
		level:     bool_to_Cchar(online),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * Print debug information via purple.
 */
func purple_debug(loglevel int, message string) {
	cmessage := C.struct_gowhatsapp_message{
		msgtype: C.char(C.gowhatsapp_message_type_log),
		level:   C.char(loglevel),
		text:    C.CString(message),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

const (
	ERROR_TRANSIENT = false
	ERROR_FATAL     = true
)

/*
 * Forward error to purple. This will cause a disconnect.
 */
func purple_error(account *PurpleAccount, message string, fatal bool) {
	level := 0
	if fatal {
		level = 1
	}
	cmessage := C.struct_gowhatsapp_message{
		account: account,
		msgtype: C.char(C.gowhatsapp_message_type_error),
		text:    C.CString(message),
		level:   C.char(level),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * Get int from the purple account's settings.
 */
func purple_get_int(account *PurpleAccount, key *C.char, default_value int) int {
	if C.gowhatsapp_account_exists(account) == 1 {
		return int(C.purple_account_get_int(account, key, C.int(default_value)))
	}
	return default_value
}

/*
 * Get bool from the purple account's settings.
 */
func purple_get_bool(account *PurpleAccount, key *C.char, default_value bool) bool {
	if C.gowhatsapp_account_exists(account) == 1 {
		return Cint_to_bool(C.purple_account_get_bool(account, key, C.int(bool_to_Cchar(default_value))))
	}
	return default_value
}

/*
 * Get string from the purple account's settings.
 */
func purple_get_string(account *PurpleAccount, key *C.char, default_value *C.char) string {
	if C.gowhatsapp_account_exists(account) == 1 {
		return C.GoString(C.purple_account_get_string(account, key, default_value))
	}
	return C.GoString(default_value)
}

/*
 * Store credential string in purple's account settings (Pidgin) and/or password (bitlbee).
 */
func purple_set_credentials(account *PurpleAccount, credentials string) {
	if C.gowhatsapp_account_exists(account) == 1 {
		C.purple_account_set_credentials(account, C.CString(credentials))
	}
}

func main() {
}
