//go:build gometa

package main

// Facebook group chats: who is in them, and their names.
//
// Who is in a group comes with the thread rows: LSAddParticipantIdToGroupThread for each member,
// LSRemoveParticipantFromThread and LSRemoveAllParticipantsForThread when they go. The handler
// keeps them by thread key, and puts them on the group's chat conversation (gometa_message_type_
// members, gometabridge.c) whenever they change and when the chat is opened (gometa_go_chat_
// joined). We are not in our own list. An encrypted group is not a group here yet (its messages
// come as a 1:1 chat's, gometa_e2ee.go).

/*
#include "gometabridge.h"
*/
import "C"

import (
	"sort"
	"strconv"
	"strings"

	"go.mau.fi/mautrix-meta/pkg/messagix/table"
)

// The group's chat conversation was opened (glue/gometa_init.c, gometabridge.c): put its members
// on it.
//
//export gometa_go_chat_joined
func gometa_go_chat_joined(account *PurpleAccount, chat *C.char) {
	if h, ok := gometaHandlers[account]; ok {
		if threadKey, err := strconv.ParseInt(C.GoString(chat), 10, 64); err == nil {
			h.pushMembers(threadKey)
		}
	}
}

// ---- members ----

// handleParticipantTables keeps who is in each group, and puts a changed group's members on its
// chat. Removals come first: a table that empties a thread's list fills it again after.
func (h *gometaHandler) handleParticipantTables(tbl *table.LSTable) {
	changed := map[int64]bool{}
	h.mu.Lock()
	if h.members == nil {
		h.members = make(map[int64]map[int64]bool)
	}
	for _, r := range tbl.LSRemoveAllParticipantsForThread {
		if r.ThreadKey != 0 {
			delete(h.members, r.ThreadKey)
			changed[r.ThreadKey] = true
		}
	}
	for _, r := range tbl.LSRemoveParticipantFromThread {
		if r.ThreadKey != 0 && r.ParticipantId != 0 {
			delete(h.members[r.ThreadKey], r.ParticipantId)
			changed[r.ThreadKey] = true
		}
	}
	for _, p := range tbl.LSAddParticipantIdToGroupThread {
		if p.ThreadKey == 0 || p.ContactId == 0 {
			continue
		}
		if h.members[p.ThreadKey] == nil {
			h.members[p.ThreadKey] = make(map[int64]bool)
		}
		h.members[p.ThreadKey][p.ContactId] = true
		changed[p.ThreadKey] = true
	}
	h.mu.Unlock()
	keys := make([]int64, 0, len(changed))
	for k := range changed {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		h.pushMembers(k)
	}
}

// groupMembers is who is in a group besides us, in fbid order.
func (h *gometaHandler) groupMembers(threadKey int64) []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []int64
	for fbid := range h.members[threadKey] {
		if fbid != h.selfID {
			out = append(out, fbid)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// pushMembers puts a group's members on its chat conversation, if it is a group.
func (h *gometaHandler) pushMembers(threadKey int64) {
	h.mu.Lock()
	isGroup := h.threadGroup[threadKey]
	h.mu.Unlock()
	if !isGroup {
		return
	}
	h.out().members(strconv.FormatInt(threadKey, 10), h.groupMembers(threadKey))
}

// gometaNotifyMembers hands a group's members ("\x1f"-joined fbids, "" for none) to the C bridge.
func gometaNotifyMembers(account *PurpleAccount, conv string, fbids []int64) {
	ids := make([]string, len(fbids))
	for i, f := range fbids {
		ids[i] = strconv.FormatInt(f, 10)
	}
	msg := C.gometa_message_t{account: account, msgtype: C.char(C.gometa_message_type_members)}
	msg.conv = C.CString(conv)
	if len(ids) > 0 {
		msg.text = C.CString(strings.Join(ids, "\x1f"))
	}
	C.gometa_process_message(msg)
}

// handleThreadNames takes a group's new name (LSSyncUpdateThreadName): the chat is named again.
func (h *gometaHandler) handleThreadNames(tbl *table.LSTable) {
	for _, n := range tbl.LSSyncUpdateThreadName {
		name := n.ThreadName
		if name == "" {
			name = n.ThreadName1
		}
		h.mu.Lock()
		isGroup := h.threadGroup[n.ThreadKey]
		h.mu.Unlock()
		if n.ThreadKey != 0 && name != "" && isGroup {
			h.notifyChat(n.ThreadKey, name)
		}
	}
}
