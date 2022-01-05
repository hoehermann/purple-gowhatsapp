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

#include <stdint.h>
#include <time.h>
//#include <unistd.h>

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

//#include "../c/constants.h"

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_session,
    gowhatsapp_message_type_contactlist_refresh,
    gowhatsapp_message_type_presence,
};

// C compatible representation of one received message.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    uintptr_t connection; /// an int representation of the purple connection pointer
    int64_t msgtype; /// message type – see above
    char *id; /// message id
    char *remoteJid; /// conversation identifier (may be a singel contact or a group)
    char *senderJid; /// message author's identifier (useful in group chats)
    char *text; /// the message payload (interpretation depends on type)
    void *blob; /// binary payload (used for inlining images)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
};
typedef struct gowhatsapp_message gowhatsapp_message_t;

extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);
extern void * gowhatsapp_get_account(uintptr_t pc);
extern int gowhatsapp_account_get_bool(void *account, const char *name, int default_value);
extern const char * gowhatsapp_account_get_string(void *account, const char *name, const char *default_value);
extern const char * gowhatsapp_account_get_password(void *account);
//extern const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account);
*/
import "C"

import (
//	"encoding/base64"
//	"encoding/json"
//	"fmt"
//	"net/http"
//	"net/url"
//	"os"
//	"strings"
//	"time"

	"github.com/Rhymen/go-whatsapp"
//	"github.com/pkg/errors"
//	"github.com/skip2/go-qrcode"
)

/*
 * Holds all data of one message which is going to be exposed
 * to the C part ("front-end") of the plug-in. However, it is converted
 * to a C compatible gowhatsapp_message type first.
 */
type MessageAggregate struct {
	info    whatsapp.MessageInfo
	text    string
	session *whatsapp.Session
	data    []byte
	err     error
	system  bool
}

/*
 * Holds all data for one account (connection) instance.
 */
type waHandler struct {
	wac                *whatsapp.Conn
	connID             C.uintptr_t
}

/*
 * This plug-in can handle multiple connections (identified by C pointer adresses).
 */
var waHandlers = make(map[C.uintptr_t]*waHandler)

//export gowhatsapp_go_login
func gowhatsapp_go_login(connID C.uintptr_t) {
    login(connID);
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(connID C.uintptr_t) {
	close(connID);
}

func main() {
}
