//go:build gometa

package main

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"google.golang.org/protobuf/proto"
)

func TestGometaIndexKeepsTheNewestMessages(t *testing.T) {
	var x gometaIndex
	for i := 0; i < gometaIndexSize+10; i++ {
		x.add(gometaIncoming{id: "m" + strconv.Itoa(i), text: "t"})
	}
	if _, ok := x.get("m9"); ok {
		t.Error("the oldest messages are kept past the bound")
	}
	if _, ok := x.get("m10"); !ok {
		t.Error("a message within the bound is gone")
	}
	if len(x.byID) != gometaIndexSize || len(x.order) != gometaIndexSize {
		t.Errorf("kept %d (%d in order), want %d", len(x.byID), len(x.order), gometaIndexSize)
	}
	x.add(gometaIncoming{id: "m10", text: "again"}) // a message shown again is not kept twice
	if len(x.order) != gometaIndexSize {
		t.Errorf("a repeated id grew the order to %d", len(x.order))
	}
	x.add(gometaIncoming{text: "no id"})
	if len(x.order) != gometaIndexSize {
		t.Error("a message without an id was kept")
	}
}

func TestGometaEditLineIsInTheMessagesChat(t *testing.T) {
	var x gometaIndex
	x.add(gometaIncoming{conv: "42", who: "7", name: "Anna", id: "mid.1", text: "helo", isGroup: true, quotedText: "q"})
	m, ok := x.editLine("mid.1", "hello", 99)
	if !ok || m.conv != "42" || m.who != "7" || m.name != "Anna" || !m.isGroup || m.text != "[EDIT] hello" || m.id != "" || m.ts != 99 || m.quotedText != "" {
		t.Errorf("got %+v ok=%v", m, ok)
	}
	if shown, _ := x.get("mid.1"); shown.text != "hello" {
		t.Errorf("the index still has %q", shown.text)
	}
	if _, ok := x.editLine("mid.unknown", "x", 1); ok {
		t.Error("an edit of a message not shown here was placed")
	}
}

func TestGometaDeleteLineQuotesTheMessage(t *testing.T) {
	var x gometaIndex
	x.add(gometaIncoming{conv: "42", who: "7", id: "mid.1", text: strings.Repeat("ä", 60)})
	m, ok := x.deleteLine("mid.1", 5)
	if want := "deleted message „" + strings.Repeat("ä", 50) + "…“."; !ok || m.text != want || m.conv != "42" || m.who != "7" {
		t.Errorf("got %+v, want text %q", m, want)
	}
	x.add(gometaIncoming{conv: "42", who: "7", id: "mid.2"})
	if m, _ := x.deleteLine("mid.2", 5); m.text != "deleted an attachment." {
		t.Errorf("got %q", m.text)
	}
	if _, ok := x.deleteLine("mid.unknown", 1); ok {
		t.Error("a deletion of a message not shown here was placed")
	}
}

func TestGometaReactionLine(t *testing.T) {
	var x gometaIndex
	x.add(gometaIncoming{conv: "42", id: "mid.1", text: "lunch?"})
	m := x.reactionLine("42", "mid.1", "7", "👍", 3)
	if m.text != "reacted with 👍 to message „lunch?“." || m.conv != "42" || m.who != "7" || m.ts != 3 {
		t.Errorf("got %+v", m)
	}
	if m := x.reactionLine("42", "mid.1", "7", "", 3); m.text != "removed their reaction to message „lunch?“." {
		t.Errorf("got %q", m.text)
	}
	if m := x.reactionLine("42", "mid.gone", "7", "❤", 3); m.text != "reacted with ❤ to an earlier message." {
		t.Errorf("got %q", m.text)
	}
}

func TestGometaQuote(t *testing.T) {
	if got := gometaQuote("a\nb"); got != "> a\n> b\n" {
		t.Errorf("got %q", got)
	}
}

func TestGometaFileName(t *testing.T) {
	if got := gometaFileName("ab", "", ".jpg"); got != "ab.jpg" {
		t.Errorf("got %q", got)
	}
	if got := gometaFileName("ab", "../x/y:z", ".pdf"); got != filepath.Join("ab", ".._x_y_z.pdf") {
		t.Errorf("a name with separators: got %q", got)
	}
	if got := gometaFileName("ab", "..", ".pdf"); got != filepath.Join("ab", "_.pdf") {
		t.Errorf("got %q", got)
	}
	h, _ := newTestGometa(t)
	h.username = "a/b@example.invalid"
	if got := h.attachmentDir(); filepath.Dir(got) != filepath.Join(h.purpleUserDir, "gometa", "attachments") {
		t.Errorf("an account name with a separator leaves the attachment folder: %q", got)
	}
}

func TestGometaE2EEReactionGoesToTheSink(t *testing.T) {
	h, r := newTestGometa(t)
	content := &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_ReactionMessage{
		ReactionMessage: &waConsumerApplication.ConsumerApplication_ReactionMessage{
			Key: &waCommon.MessageKey{ID: proto.String("7001")}, Text: proto.String("😂"),
		}}}
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "7002", content))
	if len(r.reactions) != 1 || r.reactions[0] != (gmReaction{itoa(gmPeer), "7001", itoa(gmPeer), "😂"}) || len(r.messages) != 0 {
		t.Errorf("got reactions %+v, messages %+v", r.reactions, r.messages)
	}
}

func TestGometaPlainReactionsGoToTheSink(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{
		LSUpsertReaction: []*table.LSUpsertReaction{{ThreadKey: gmPeer, MessageId: "mid.1", ActorId: gmPeer, Reaction: "👍"}},
		LSDeleteReaction: []*table.LSDeleteReaction{{ThreadKey: gmPeer, MessageId: "mid.2", ActorId: gmPeer}},
	})
	want := []gmReaction{{itoa(gmPeer), "mid.1", itoa(gmPeer), "👍"}, {itoa(gmPeer), "mid.2", itoa(gmPeer), ""}}
	if len(r.reactions) != 2 || r.reactions[0] != want[0] || r.reactions[1] != want[1] {
		t.Errorf("got %+v, want %+v", r.reactions, want)
	}
}

func TestGometaOwnPlainMessageCanBeQuoted(t *testing.T) {
	h, r := newTestGometa(t)
	r.confirm = true
	h.sendMessage(itoa(gmPeer), "see you")
	m := h.index.reactionLine(itoa(gmPeer), "mid.$sent", itoa(gmPeer), "👍", 1)
	if m.text != "reacted with 👍 to message „see you“." {
		t.Errorf("got %q", m.text)
	}
}

func TestGometaOwnEncryptedMessageCanBeQuoted(t *testing.T) {
	h, r := newTestGometa(t)
	h.e2eeContacts[gmPeer] = true
	h.sendMessage(itoa(gmPeer), "see you")
	if len(r.sentFB) != 1 {
		t.Fatalf("sent %d encrypted messages", len(r.sentFB))
	}
	m := h.index.reactionLine(itoa(gmPeer), string(r.sentFB[0].id), itoa(gmPeer), "👍", 1)
	if m.text != "reacted with 👍 to message „see you“." {
		t.Errorf("got %q", m.text)
	}
}

func TestGometaFailedEncryptedSendIsNotQuotable(t *testing.T) {
	h, r := newTestGometa(t)
	h.e2eeContacts[gmPeer] = true
	r.sendErr = errors.New("offline")
	h.sendMessage(itoa(gmPeer), "lost")
	if len(h.index.order) != 0 || len(r.failures) != 1 {
		t.Errorf("index %v, failures %v", h.index.order, r.failures)
	}
}

func TestGometaE2EEStoreStaysInTheUserDir(t *testing.T) {
	if got := gometaE2EEStorePath("/u", "me@example.invalid"); got != "/u/gometa-e2ee-me@example.invalid.db" {
		t.Errorf("an e-mail address is changed: %q", got)
	}
	if got := gometaE2EEStorePath("/u", "../../x/y"); filepath.Dir(got) != "/u" {
		t.Errorf("a username with separators leaves the user directory: %q", got)
	}
	if got := gometaE2EEStoreURI("/a%b/c#d.db"); got != "file:/a%25b/c%23d.db" {
		t.Errorf("got %q", got)
	}
}
