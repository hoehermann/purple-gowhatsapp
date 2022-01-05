/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2019 Hermann Höhne
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
#cgo CFLAGS: -DCGO

#include <stdint.h> // for int64_t
#include <time.h> // for time_t

#ifdef _MSC_VER
#ifdef GO_CGO_PROLOGUE_H
#error GO_CGO_PROLOGUE_H already defined, but must override.
#endif
#define GO_CGO_PROLOGUE_H
typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt32 GoInt;
typedef GoUint32 GoUint;
typedef size_t GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
typedef char _check_for_32_bit_pointer_matching_GoInt[sizeof(void*)==32/8 ? 1:-1];
typedef _GoString_ GoString;
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;
#endif

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_session,
    gowhatsapp_message_type_contactlist_refresh,
    gowhatsapp_message_type_presence,
};

// Structure to communicate go → purple.
// This holds all data for incoming messages, error messages, login data, etc.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    char *username; /// username identifying the account
    char *id; /// message id
    char *remoteJid; /// conversation identifier (may be a single contact or a group)
    char *senderJid; /// message author's identifier (useful in group chats)
    char *text; /// the message payload (interpretation depends on type)
    void *blob; /// binary payload (used for inlining images)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char msgtype; /// message type – see above
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
};
typedef struct gowhatsapp_message gowhatsapp_message_t;

// for feeding messages from go into purple
extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);
*/
import "C"

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

/*
 * Holds all data for one connection.
 */
type Client struct {
	client   *whatsmeow.Client
	username string
}

/*
 * This plug-in can handle multiple connections (identified by JID).
 */
var clients = make(map[string]*Client)

/*
 * whatsmew stores all data in a container (set by main.init()).
 */
var container *sqlstore.Container = nil

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

func main() {
}
