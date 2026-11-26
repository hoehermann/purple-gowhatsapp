//go:build gometa

package main

import (
	"reflect"
	"testing"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func gmText(text string) *waConsumerApplication.ConsumerApplication_Content {
	return &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_MessageText{
		MessageText: &waCommon.MessageText{Text: proto.String(text)}}}
}

// ---- typing sent ----

func TestGometaTypingInAnEncryptedChatGoesAsChatPresence(t *testing.T) {
	h, r := newTestGometa(t)
	h.e2eeContacts[gmPeer] = true
	h.sendTyping(itoa(gmPeer), true)
	h.sendTyping(itoa(gmPeer), false)
	want := []gmPresence{{gmJID(gmPeer), true}, {gmJID(gmPeer), false}}
	if !reflect.DeepEqual(r.presences, want) || len(r.tasks) != 0 {
		t.Errorf("presences %+v tasks %+v, want %+v", r.presences, r.tasks, want)
	}
}

func TestGometaTypingInAPlainChatIsAPresenceTask(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{LSDeleteThenInsertThread: []*table.LSDeleteThenInsertThread{{ThreadKey: gmPeer, ThreadType: table.ONE_TO_ONE}}})
	h.sendTyping(itoa(gmPeer), true)
	h.sendTyping(itoa(gmPeer), false)
	if len(r.tasks) != 2 || len(r.presences) != 0 {
		t.Fatalf("tasks %+v presences %+v", r.tasks, r.presences)
	}
	on, off := r.tasks[0], r.tasks[1]
	want := &socket.UpdatePresenceTask{ThreadKey: gmPeer, IsTyping: 1, SyncGroup: 1, ThreadType: int64(table.ONE_TO_ONE)}
	if !reflect.DeepEqual(on.task, want) || !on.stateless {
		t.Errorf("typing: got %+v (stateless %v), want %+v", on.task, on.stateless, want)
	}
	if off.task.(*socket.UpdatePresenceTask).IsTyping != 0 {
		t.Errorf("stopped: got %+v", off.task)
	}
}

func TestGometaTypingIgnoresTheSignInChat(t *testing.T) {
	h, r := newTestGometa(t)
	h.sendTyping("Facebook", true)
	if len(r.tasks)+len(r.presences) != 0 {
		t.Errorf("sent %+v %+v", r.tasks, r.presences)
	}
}

// ---- typing received ----

func TestGometaTypingFromAPlainChat(t *testing.T) {
	h, r := newTestGometa(t)
	h.threadGroup[gmGroup] = true
	h.parseTable(&table.LSTable{LSUpdateTypingIndicator: []*table.LSUpdateTypingIndicator{
		{ThreadKey: gmPeer, SenderId: gmPeer, IsTyping: true},
		{ThreadKey: gmPeer, SenderId: gmSelf, IsTyping: true}, // ours, from another device
		{ThreadKey: gmGroup, SenderId: gmPeer2, IsTyping: true},
		{ThreadKey: gmPeer, SenderId: gmPeer, IsTyping: false},
	}})
	want := []gmTyping{{itoa(gmPeer), true}, {itoa(gmPeer), false}}
	if !reflect.DeepEqual(r.typings, want) {
		t.Errorf("got %+v, want %+v", r.typings, want)
	}
}

func TestGometaTypingFromAnEncryptedChat(t *testing.T) {
	h, r := newTestGometa(t)
	src := waTypes.MessageSource{Chat: gmJID(gmPeer), Sender: gmJID(gmPeer)}
	h.e2eeEventHandler(&events.ChatPresence{MessageSource: src, State: waTypes.ChatPresenceComposing})
	h.e2eeEventHandler(&events.ChatPresence{MessageSource: src, State: waTypes.ChatPresencePaused})
	group := waTypes.MessageSource{Chat: waTypes.JID{User: itoa(gmGroup), Server: waTypes.GroupServer}, Sender: gmJID(gmPeer2), IsGroup: true}
	h.e2eeEventHandler(&events.ChatPresence{MessageSource: group, State: waTypes.ChatPresenceComposing})
	want := []gmTyping{{itoa(gmPeer), true}, {itoa(gmPeer), false}}
	if !reflect.DeepEqual(r.typings, want) {
		t.Errorf("got %+v, want %+v", r.typings, want)
	}
}

// ---- read receipts sent ----

func TestGometaShownEncryptedChatIsMarkedRead(t *testing.T) {
	h, r := newTestGometa(t)
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "8001", gmText("one")))
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "8002", gmText("two")))
	h.markRead(itoa(gmPeer))
	want := []gmMarkRead{{[]waTypes.MessageID{"8001", "8002"}, gmJID(gmPeer), gmJID(gmPeer)}}
	if !reflect.DeepEqual(r.markReads, want) {
		t.Fatalf("got %+v, want %+v", r.markReads, want)
	}
	// Messenger's apps read the chat's Facebook thread: a 1:1 one is under the partner's fbid.
	wantTask := &socket.ThreadMarkReadTask{ThreadId: gmPeer, LastReadWatermarkTs: 1700000000999, SyncGroup: 95}
	if len(r.tasks) != 1 || !reflect.DeepEqual(r.tasks[0].task, wantTask) || r.tasks[0].stateless {
		t.Fatalf("Facebook thread: got %+v, want %+v", r.tasks, wantTask)
	}
	h.markRead(itoa(gmPeer)) // nothing new: no second receipt
	if len(r.markReads) != 1 || len(r.tasks) != 1 {
		t.Errorf("again: %+v %+v", r.markReads, r.tasks)
	}
}

func TestGometaEncryptedChatReadsTheFacebookThreadFacebookNamed(t *testing.T) {
	h, r := newTestGometa(t)
	const fbKey = int64(7000000000000007)
	h.parseTable(&table.LSTable{LSVerifyHybridThreadExists: []*table.LSVerifyHybridThreadExists{
		{ThreadKey: fbKey, ThreadJID: gmPeer, ThreadType: table.ENCRYPTED_OVER_WA_ONE_TO_ONE}}})
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "8101", gmText("one")))
	h.markRead(itoa(gmPeer))
	want := &socket.ThreadMarkReadTask{ThreadId: fbKey, LastReadWatermarkTs: 1700000000999, SyncGroup: 95}
	if len(r.tasks) != 1 || !reflect.DeepEqual(r.tasks[0].task, want) {
		t.Errorf("got %+v, want %+v", r.tasks, want)
	}
	// Facebook's other name for the mapping gives the same.
	h2, r2 := newTestGometa(t)
	h2.parseTable(&table.LSTable{LSUpdateThreadAuthorityAndMappingWithOTIDFromJID: []*table.LSUpdateThreadAuthorityAndMappingWithOTIDFromJID{
		{ThreadJID: gmPeer, ThreadKey: fbKey}}})
	h2.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "8102", gmText("two")))
	h2.markRead(itoa(gmPeer))
	if len(r2.tasks) != 1 || r2.tasks[0].task.(*socket.ThreadMarkReadTask).ThreadId != fbKey {
		t.Errorf("authority mapping: got %+v", r2.tasks)
	}
}

func TestGometaOwnEncryptedMessageReadsOnlyTheFacebookThread(t *testing.T) {
	h, r := newTestGometa(t)
	mine := gmFBMessage(gmJID(gmPeer), gmJID(gmSelf), "8201", gmText("from my phone"))
	mine.Info.IsFromMe = true
	mine.Info.Timestamp = time.Unix(1700000100, 250*int64(time.Millisecond))
	h.handleE2EEMessage(mine)
	h.markRead(itoa(gmPeer))
	if len(r.markReads) != 0 {
		t.Errorf("a receipt for our own message: %+v", r.markReads)
	}
	want := &socket.ThreadMarkReadTask{ThreadId: gmPeer, LastReadWatermarkTs: 1700000100250, SyncGroup: 95}
	if len(r.tasks) != 1 || !reflect.DeepEqual(r.tasks[0].task, want) {
		t.Errorf("got %+v, want %+v", r.tasks, want)
	}
}

func TestGometaEncryptedGroupReceiptsGoPerSender(t *testing.T) {
	h, r := newTestGometa(t)
	group := waTypes.JID{User: itoa(gmGroup), Server: waTypes.GroupServer}
	h.handleE2EEMessage(gmFBMessage(group, gmJID(gmPeer), "9001", gmText("a")))
	h.handleE2EEMessage(gmFBMessage(group, gmJID(gmPeer2), "9002", gmText("b")))
	h.handleE2EEMessage(gmFBMessage(group, gmJID(gmPeer), "9003", gmText("c")))
	h.markRead(itoa(gmGroup))
	want := []gmMarkRead{
		{[]waTypes.MessageID{"9001", "9003"}, group, gmJID(gmPeer)},
		{[]waTypes.MessageID{"9002"}, group, gmJID(gmPeer2)},
	}
	if !reflect.DeepEqual(r.markReads, want) {
		t.Errorf("got %+v, want %+v", r.markReads, want)
	}
	// Facebook named no thread for the group: none is guessed.
	if len(r.tasks) != 0 {
		t.Errorf("a Facebook thread guessed for an encrypted group: %+v", r.tasks)
	}
}

func TestGometaShownPlainChatIsMarkedRead(t *testing.T) {
	h, r := newTestGometa(t)
	older, newer := gmInsert("mid.a", "a", gmPeer), gmInsert("mid.b", "b", gmPeer)
	newer.TimestampMs = older.TimestampMs + 5000
	mine := gmInsert("mid.c", "c", gmSelf) // ours from another device: not a reason for a receipt
	mine.TimestampMs = older.TimestampMs + 9000
	h.parseTable(&table.LSTable{LSInsertMessage: []*table.LSInsertMessage{newer, older, mine}})
	h.markRead(itoa(gmPeer))
	want := &socket.ThreadMarkReadTask{ThreadId: gmPeer, LastReadWatermarkTs: newer.TimestampMs, SyncGroup: 1}
	if len(r.tasks) != 1 || !reflect.DeepEqual(r.tasks[0].task, want) || r.tasks[0].stateless {
		t.Fatalf("got %+v, want %+v", r.tasks, want)
	}
	if len(r.markReads) != 0 {
		t.Errorf("encrypted receipts for a plain chat: %+v", r.markReads)
	}
	h.markRead(itoa(gmPeer))
	if len(r.tasks) != 1 {
		t.Errorf("a second receipt with nothing new: %+v", r.tasks)
	}
}
