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
#include "msc_adjustments.h"

// for feeding messages from go into purple
extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);
*/
import "C"

import "time"

//export gowhatsapp_go_init
func gowhatsapp_go_init(purple_user_dir *C.char) C.int {
	return C.int(init_(C.GoString(purple_user_dir)))
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(username *C.char) {
	login(C.GoString(username))
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(username *C.char) {
	close(C.GoString(username))
}

/*
 * This will display a QR code via PurpleRequest API.
 */
func purple_display_qrcode(username string, qr_data string, png []byte) {
	cmessage := C.struct_gowhatsapp_message{
		username: C.CString(username),
		msgtype:  C.char(C.gowhatsapp_message_type_login),
		text:     C.CString(qr_data),
		blob:     C.CBytes(png),
		blobsize: C.size_t(len(png)),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

/*
 * This will display a text message.
 * Single participants and group chats.
 */
func purple_display_text_message(username string, id string, remoteJid string, timestamp time.Time, text string) {
	cmessage := C.struct_gowhatsapp_message{
		username:  C.CString(username),
		msgtype:   C.char(C.gowhatsapp_message_type_text),
		id:        C.CString(id),
		remoteJid: C.CString(remoteJid),
		timestamp: C.time_t(timestamp.Unix()),
		text:      C.CString(text),
	}
	C.gowhatsapp_process_message_bridge(cmessage)
}

func main() {
}