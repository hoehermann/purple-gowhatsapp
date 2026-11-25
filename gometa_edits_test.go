//go:build gometa

package main

import (
	"reflect"
	"testing"

	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func gmEditContent(target, text string) *waConsumerApplication.ConsumerApplication_Content {
	return &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_EditMessage{
		EditMessage: &waConsumerApplication.ConsumerApplication_EditMessage{
			Key:     &waCommon.MessageKey{ID: proto.String(target)},
			Message: &waCommon.MessageText{Text: proto.String(text)},
		}}}
}

func gmRevoke(chat, sender waTypes.JID, id, target string, fromMe bool) *events.FBMessage {
	evt := gmFBMessage(chat, sender, id, nil)
	evt.Info.IsFromMe = fromMe
	evt.Message = &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{
		Payload: &waConsumerApplication.ConsumerApplication_Payload_ApplicationData{ApplicationData: &waConsumerApplication.ConsumerApplication_ApplicationData{
			ApplicationContent: &waConsumerApplication.ConsumerApplication_ApplicationData_Revoke{Revoke: &waConsumerApplication.ConsumerApplication_RevokeMessage{
				Key: &waCommon.MessageKey{ID: proto.String(target)}}},
		}},
	}}
	return evt
}

// ---- encrypted chats ----

func TestGometaE2EEEditChangesTheMessageByItsStoredID(t *testing.T) {
	h, r := newTestGometa(t)
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "6001", gmText("helo")))
	stored := onlyMessage(t, r).id
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "6002", gmEditContent("6001", "hello")))
	if want := []gmEdit{{stored, "hello"}}; !reflect.DeepEqual(r.edits, want) {
		t.Errorf("edits %+v, want %+v", r.edits, want)
	}
	if len(r.messages) != 1 {
		t.Errorf("the edit was shown as a message of its own: %+v", r.messages)
	}
	if h.e2eeMsgMeta["6001"].text != "hello" {
		t.Errorf("a reply would still quote %q", h.e2eeMsgMeta["6001"].text)
	}
}

func TestGometaE2EERevokeDeletesTheMessage(t *testing.T) {
	h, r := newTestGometa(t)
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "6101", gmText("oops")))
	h.handleE2EEMessage(gmRevoke(gmJID(gmPeer), gmJID(gmPeer), "6102", "6101", false))
	if want := []string{"6101"}; !reflect.DeepEqual(r.removes, want) || len(r.messages) != 1 {
		t.Errorf("removes %+v messages %d", r.removes, len(r.messages))
	}
}

// Someone else may not change a message they did not send; we may change our own.
func TestGometaE2EEOnlyTheSenderMayChangeAMessage(t *testing.T) {
	h, r := newTestGometa(t)
	group := waTypes.JID{User: itoa(gmGroup), Server: waTypes.GroupServer}
	h.handleE2EEMessage(gmFBMessage(group, gmJID(gmPeer), "6201", gmText("mine to change")))
	h.handleE2EEMessage(gmFBMessage(group, gmJID(gmPeer2), "6202", gmEditContent("6201", "forged")))
	h.handleE2EEMessage(gmRevoke(group, gmJID(gmPeer2), "6203", "6201", false))
	if len(r.edits)+len(r.removes) != 0 {
		t.Fatalf("a forged change went through: %+v %+v", r.edits, r.removes)
	}
	// our own message, sent from here, deleted from our other device
	h.e2eeMsgMeta["6300"] = e2eeMsgInfo{fromMe: true, sender: itoa(gmSelf)}
	h.handleE2EEMessage(gmRevoke(gmJID(gmPeer), gmJID(gmSelf), "6301", "6300", true))
	if want := []string{"6300"}; !reflect.DeepEqual(r.removes, want) {
		t.Errorf("own delete: %+v", r.removes)
	}
	// the other side may not delete ours, and we may not delete theirs
	h.e2eeMsgMeta["6310"] = e2eeMsgInfo{fromMe: true, sender: itoa(gmSelf)}
	h.handleE2EEMessage(gmRevoke(gmJID(gmPeer), gmJID(gmPeer), "6311", "6310", false))
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "6320", gmText("theirs")))
	h.handleE2EEMessage(gmRevoke(gmJID(gmPeer), gmJID(gmSelf), "6321", "6320", true))
	if want := []string{"6300"}; !reflect.DeepEqual(r.removes, want) {
		t.Errorf("a delete crossed sides: %+v", r.removes)
	}
}

// After a restart the cache is empty: the change still goes, by id.
func TestGometaE2EEEditOfAMessageFromBeforeTheRestart(t *testing.T) {
	h, r := newTestGometa(t)
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "6402", gmEditContent("6401", "new")))
	h.handleE2EEMessage(gmRevoke(gmJID(gmPeer), gmJID(gmPeer), "6403", "6400", false))
	if !reflect.DeepEqual(r.edits, []gmEdit{{"6401", "new"}}) || !reflect.DeepEqual(r.removes, []string{"6400"}) {
		t.Errorf("edits %+v removes %+v", r.edits, r.removes)
	}
}

// ---- plain chats ----

func TestGometaPlainEditAndDelete(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{LSInsertMessage: []*table.LSInsertMessage{gmInsert("mid.e", "helo", gmPeer)}})
	stored := onlyMessage(t, r).id
	h.parseTable(&table.LSTable{LSEditMessage: []*table.LSEditMessage{{MessageID: "mid.e", Text: "hello", EditCount: 1}}})
	h.parseTable(&table.LSTable{LSDeleteMessage: []*table.LSDeleteMessage{{ThreadKey: gmPeer, MessageId: "mid.e"}}})
	if !reflect.DeepEqual(r.edits, []gmEdit{{stored, "hello"}}) || !reflect.DeepEqual(r.removes, []string{stored}) {
		t.Errorf("edits %+v removes %+v", r.edits, r.removes)
	}
}

func TestGometaPlainUnsendIsADelete(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{LSDeleteThenInsertMessage: []*table.LSDeleteThenInsertMessage{
		{ThreadKey: gmPeer, MessageId: "mid.u", IsUnsent: true},
		{ThreadKey: gmPeer, MessageId: "mid.v", IsUnsent: false}, // not an unsend: left alone
	}})
	unsent := gmInsert("mid.w", "", gmPeer)
	unsent.IsUnsent = true
	h.parseTable(&table.LSTable{LSInsertMessage: []*table.LSInsertMessage{unsent}})
	if want := []string{"mid.u", "mid.w"}; !reflect.DeepEqual(r.removes, want) || len(r.messages) != 0 {
		t.Errorf("removes %+v messages %+v, want %+v", r.removes, r.messages, want)
	}
}

// Deleted and inserted again in one table: replaced, not deleted.
func TestGometaPlainDeleteThenInsertIsNoDelete(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{
		LSDeleteMessage: []*table.LSDeleteMessage{{ThreadKey: gmPeer, MessageId: "mid.r"}},
		LSInsertMessage: []*table.LSInsertMessage{gmInsert("mid.r", "again", gmPeer)},
	})
	if len(r.removes) != 0 || len(r.messages) != 1 {
		t.Errorf("removes %+v messages %+v", r.removes, r.messages)
	}
}
