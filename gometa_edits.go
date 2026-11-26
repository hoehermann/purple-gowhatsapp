//go:build gometa

package main

// A Facebook message edited or deleted for everyone after it was shown. Both name the message by
// its id:
//   encrypted chats: the whatsmeow message id (evt.Info.ID; for our own sends from here, the id we
//                    sent with). An edit's or a revoke's MessageKey.ID names it.
//   plain chats:     the Facebook MessageId ("mid.$..."; for our own sends from here, the one
//                    LSReplaceOptimsiticMessage gives). The edit and delete tables name it.
// The sink shows the change (gometa_sink.go).

import (
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/types/events"
)

// e2eeMayChange tells whether whoever sent evt may edit or delete the message stored under target.
// The per-message cache (e2eeMsgMeta) knows who sent each encrypted message seen since the plugin
// started, and an edit or delete from anyone else is dropped. A message from before that is not in
// the cache: its sender cannot be checked here, and the change is passed on by id as it came.
func (h *gometaHandler) e2eeMayChange(evt *events.FBMessage, target string) bool {
	h.mu.Lock()
	meta, known := h.e2eeMsgMeta[target]
	h.mu.Unlock()
	if !known {
		return true
	}
	if meta.fromMe || evt.Info.IsFromMe {
		return meta.fromMe && evt.Info.IsFromMe
	}
	return meta.sender == evt.Info.Sender.User
}

// handleE2EEEdit takes an encrypted edit: the message under its key's id now reads the new text.
func (h *gometaHandler) handleE2EEEdit(evt *events.FBMessage, key *waCommon.MessageKey, text string) {
	target := key.GetID()
	if target == "" || text == "" || !h.e2eeMayChange(evt, target) {
		return
	}
	h.mu.Lock()
	if meta, ok := h.e2eeMsgMeta[target]; ok {
		meta.text = text // a later reply quotes the new text
		h.e2eeMsgMeta[target] = meta
	}
	h.mu.Unlock()
	h.out().edit(target, text)
}

// handleE2EERevoke takes an encrypted "delete for everyone".
func (h *gometaHandler) handleE2EERevoke(evt *events.FBMessage, key *waCommon.MessageKey) {
	target := key.GetID()
	if target == "" || !h.e2eeMayChange(evt, target) {
		return
	}
	h.out().remove(target)
}

// handleEditTables takes a plain chat's edits and deletes. A message deleted and inserted again in
// the same table was replaced, not deleted (mautrix-meta skips such a thread delete too), so only a
// delete without an insert counts; LSDeleteThenInsertMessage is a delete only when it says the
// message was unsent, as mautrix-meta reads it.
func (h *gometaHandler) handleEditTables(tbl *table.LSTable) {
	for _, e := range tbl.LSEditMessage {
		if e.MessageID != "" && e.Text != "" {
			h.out().edit(e.MessageID, e.Text)
		}
	}
	inserted := make(map[string]bool, len(tbl.LSInsertMessage)+len(tbl.LSUpsertMessage))
	for _, m := range tbl.LSInsertMessage {
		inserted[m.MessageId] = !m.IsUnsent
	}
	for _, m := range tbl.LSUpsertMessage {
		inserted[m.MessageId] = !m.IsUnsent
	}
	removed := make(map[string]bool)
	remove := func(id string) {
		if id != "" && !inserted[id] && !removed[id] {
			removed[id] = true
			h.out().remove(id)
		}
	}
	for _, d := range tbl.LSDeleteMessage {
		remove(d.MessageId)
	}
	for _, d := range tbl.LSDeleteThenInsertMessage {
		if d.IsUnsent {
			remove(d.MessageId)
		}
	}
	// A message that arrives already unsent (a later sync of one that was shown) is a delete too.
	for _, m := range tbl.LSInsertMessage {
		if m.IsUnsent {
			remove(m.MessageId)
		}
	}
	for _, m := range tbl.LSUpsertMessage {
		if m.IsUnsent {
			remove(m.MessageId)
		}
	}
}
