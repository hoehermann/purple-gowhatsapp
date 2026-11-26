//go:build gometa

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow"
	armadillo "go.mau.fi/whatsmeow/proto"
	"go.mau.fi/whatsmeow/proto/waArmadilloApplication"
	"go.mau.fi/whatsmeow/proto/waArmadilloXMA"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	"go.mau.fi/whatsmeow/proto/waMsgApplication"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// ---- a handler with a recording sink and a fake wire ----

type gmEdit struct{ id, text string }
type gmAttachment struct {
	m              gometaIncoming
	path, mimetype string
}
type gmReaction struct{ conv, target, actor, emoji string }
type gmTyping struct {
	who string
	on  bool
}
type gmPresence struct {
	to        waTypes.JID
	composing bool
}
type gmMarkRead struct {
	ids          []waTypes.MessageID
	chat, sender waTypes.JID
}
type gmNotice struct {
	conv    string
	isGroup bool
	text    string
	ts      int64
}
type gmTask struct {
	task      socket.Task
	stateless bool
}

type gmRecorder struct {
	mu       sync.Mutex
	messages []gometaIncoming
	edits    []gmEdit
	removes  []string
	typings  []gmTyping
	notices  []gmNotice
	lists    map[string][]int64 // the last list put on each group chat
	chats    []gmChat
	answer   func(socket.Task) *table.LSTable // what a task other than a send answers
	taskErr  error                            // what such a task returns

	presences []gmPresence
	markReads []gmMarkRead
	tasks     []gmTask
	fetched   []string
	media     map[string][]byte // what a download or fetch returns, by URL / direct path
	failNext  error

	attachments []gmAttachment
	reactions   []gmReaction

	failures []string   // errors shown to the user
	sentFB   []gmSentFB // encrypted sends
	sendErr  error      // what an encrypted send returns
	sent     []*socket.SendMessageTask
	confirm  bool // a plain send is confirmed (LSReplaceOptimsiticMessage names its otid)
}

type gmSentFB struct {
	to  waTypes.JID
	msg armadillo.RealMessageApplicationSub
	id  waTypes.MessageID
}

func (r *gmRecorder) buddy(int64, string) {}
func (r *gmRecorder) message(m gometaIncoming) {
	r.mu.Lock()
	r.messages = append(r.messages, m)
	r.mu.Unlock()
}
func (r *gmRecorder) attachment(m gometaIncoming, path, mimetype string) {
	r.mu.Lock()
	r.attachments = append(r.attachments, gmAttachment{m, path, mimetype})
	r.mu.Unlock()
}
func (r *gmRecorder) reaction(conv, target, actor, emoji string) {
	r.reactions = append(r.reactions, gmReaction{conv, target, actor, emoji})
}
func (r *gmRecorder) edit(id, text string) { r.edits = append(r.edits, gmEdit{id, text}) }
func (r *gmRecorder) remove(id string)     { r.removes = append(r.removes, id) }
func (r *gmRecorder) typing(who string, on bool) {
	r.typings = append(r.typings, gmTyping{who, on})
}

func (r *gmRecorder) failed(text string, _ bool) { r.failures = append(r.failures, text) }

type gmChat struct {
	key  int64
	name string
}

func (r *gmRecorder) chat(key int64, name string) { r.chats = append(r.chats, gmChat{key, name}) }
func (r *gmRecorder) members(conv string, fbids []int64) {
	if r.lists == nil {
		r.lists = map[string][]int64{}
	}
	r.lists[conv] = append([]int64{}, fbids...)
}
func (r *gmRecorder) notice(conv string, isGroup bool, text string, ts int64) {
	r.notices = append(r.notices, gmNotice{conv, isGroup, text, ts})
}
func (r *gmRecorder) sendFB(_ context.Context, to waTypes.JID, msg armadillo.RealMessageApplicationSub, _ *waMsgApplication.MessageApplication_Metadata, id waTypes.MessageID) error {
	if r.sendErr != nil {
		return r.sendErr
	}
	r.sentFB = append(r.sentFB, gmSentFB{to, msg, id})
	return nil
}
func (r *gmRecorder) sendTask(_ context.Context, task socket.Task) (*table.LSTable, error) {
	t, ok := task.(*socket.SendMessageTask)
	if !ok {
		r.tasks = append(r.tasks, gmTask{task, false})
		if r.taskErr != nil {
			return nil, r.taskErr
		}
		if r.answer != nil {
			return r.answer(task), nil
		}
		return &table.LSTable{}, nil
	}
	r.sent = append(r.sent, t)
	if !r.confirm {
		return &table.LSTable{}, nil
	}
	return &table.LSTable{LSReplaceOptimsiticMessage: []*table.LSReplaceOptimsiticMessage{
		{OfflineThreadingId: strconv.FormatInt(t.Otid, 10), MessageId: "mid.$sent"}}}, nil
}
func (r *gmRecorder) waitSend(context.Context) error { return nil }
func (r *gmRecorder) chatPresence(_ context.Context, to waTypes.JID, composing bool) error {
	r.presences = append(r.presences, gmPresence{to, composing})
	return nil
}
func (r *gmRecorder) markRead(_ context.Context, ids []waTypes.MessageID, _ time.Time, chat, sender waTypes.JID) error {
	r.markReads = append(r.markReads, gmMarkRead{append([]waTypes.MessageID(nil), ids...), chat, sender})
	return nil
}
func (r *gmRecorder) downloadFB(_ context.Context, integral *waMediaTransport.WAMediaTransport_Integral, _ whatsmeow.MediaType) ([]byte, error) {
	return r.media[integral.GetDirectPath()], nil
}
func (r *gmRecorder) fbTask(_ context.Context, task socket.Task, stateless bool) error {
	r.tasks = append(r.tasks, gmTask{task, stateless})
	return nil
}
func (r *gmRecorder) fetch(_ context.Context, url, _ string) ([]byte, error) {
	r.fetched = append(r.fetched, url)
	if r.failNext != nil {
		err := r.failNext
		r.failNext = nil
		return nil, err
	}
	return r.media[url], nil
}

const gmSelf, gmPeer, gmPeer2, gmGroup = int64(100000000000001), int64(100000000000002), int64(100000000000003), int64(900000000000009)

func newTestGometa(t *testing.T) (*gometaHandler, *gmRecorder) {
	t.Helper()
	prevGo := gometaGo
	gometaGo = func(f func()) { f() }
	t.Cleanup(func() { gometaGo = prevGo })
	r := &gmRecorder{media: map[string][]byte{}}
	h := &gometaHandler{
		logger: zerolog.Nop(), ctx: context.Background(), selfID: gmSelf,
		username: "test@example.invalid", purpleUserDir: t.TempDir(),
		threadGroup: map[int64]bool{}, contactName: map[int64]string{}, sentOtids: map[int64]bool{},
		e2eeContacts: map[int64]bool{}, e2eeMsgMeta: map[string]e2eeMsgInfo{},
		members: map[int64]map[int64]bool{},
		sink:    r, wire: r,
	}
	h.initParity()
	return h, r
}

func gmFBMessage(chat, sender waTypes.JID, id string, content *waConsumerApplication.ConsumerApplication_Content) *events.FBMessage {
	return &events.FBMessage{
		Info: waTypes.MessageInfo{
			MessageSource: waTypes.MessageSource{Chat: chat, Sender: sender},
			ID:            waTypes.MessageID(id), Timestamp: time.Unix(1700000000, 0),
		},
		Message: &waConsumerApplication.ConsumerApplication{Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{Content: content},
		}},
	}
}

func gmJID(fbid int64) waTypes.JID {
	return waTypes.JID{User: itoa(fbid), Server: waTypes.MessengerServer}
}

func itoa(i int64) string { return strconv.FormatInt(i, 10) }

func gmTransport(directPath, mime string, sha []byte) *waMediaTransport.WAMediaTransport {
	return &waMediaTransport.WAMediaTransport{
		Integral: &waMediaTransport.WAMediaTransport_Integral{
			FileSHA256: sha, MediaKey: []byte("k"), FileEncSHA256: []byte("e"), DirectPath: proto.String(directPath),
		},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{Mimetype: proto.String(mime), FileLength: proto.Uint64(4)},
	}
}

func gmSticker(t *testing.T, directPath, mime string, sha []byte) *waConsumerApplication.ConsumerApplication_Content {
	t.Helper()
	sm := &waConsumerApplication.ConsumerApplication_StickerMessage{}
	if err := sm.Set(&waMediaTransport.StickerTransport{Integral: &waMediaTransport.StickerTransport_Integral{Transport: gmTransport(directPath, mime, sha)}}); err != nil {
		t.Fatal(err)
	}
	return &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_StickerMessage{StickerMessage: sm}}
}

func gmImage(t *testing.T, directPath string, sha []byte, caption string) *waConsumerApplication.ConsumerApplication_Content {
	t.Helper()
	im := &waConsumerApplication.ConsumerApplication_ImageMessage{}
	if caption != "" {
		im.Caption = &waCommon.MessageText{Text: proto.String(caption)}
	}
	if err := im.Set(&waMediaTransport.ImageTransport{Integral: &waMediaTransport.ImageTransport_Integral{Transport: gmTransport(directPath, "image/jpeg", sha)}}); err != nil {
		t.Fatal(err)
	}
	return &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_ImageMessage{ImageMessage: im}}
}

func onlyAttachment(t *testing.T, r *gmRecorder) gmAttachment {
	t.Helper()
	if len(r.attachments) != 1 || len(r.messages) != 0 {
		t.Fatalf("got %d attachments and %d messages, want 1 attachment: %+v %+v", len(r.attachments), len(r.messages), r.attachments, r.messages)
	}
	return r.attachments[0]
}

func onlyMessage(t *testing.T, r *gmRecorder) gometaIncoming {
	t.Helper()
	if len(r.messages) != 1 {
		t.Fatalf("got %d messages, want 1: %+v", len(r.messages), r.messages)
	}
	return r.messages[0]
}

// ---- encrypted chats ----

func TestGometaE2EETextReadsAnExtendedText(t *testing.T) {
	c := &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_ExtendedTextMessage{
		ExtendedTextMessage: &waConsumerApplication.ConsumerApplication_ExtendedTextMessage{
			Text:  &waCommon.MessageText{Text: proto.String("look https://example.org/a")},
			Title: proto.String("Example"), Description: proto.String("A page"),
		}}}
	if got, want := gometaE2EEText(c), "Example\nA page\nlook https://example.org/a"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGometaE2EETextReadsALocation(t *testing.T) {
	c := &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_LocationMessage{
		LocationMessage: &waConsumerApplication.ConsumerApplication_LocationMessage{
			Location: &waConsumerApplication.ConsumerApplication_Location{DegreesLatitude: proto.Float64(52.37), DegreesLongitude: proto.Float64(4.89), Name: proto.String("Dam")},
			Address:  proto.String("Amsterdam"),
		}}}
	want := "[LOCATION] Dam\nAmsterdam\n52.370000, 4.890000\nhttps://www.openstreetmap.org/?mlat=52.370000&mlon=4.890000"
	if got := gometaE2EEText(c); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	live := &waConsumerApplication.ConsumerApplication_Content{Content: &waConsumerApplication.ConsumerApplication_Content_LiveLocationMessage{
		LiveLocationMessage: &waConsumerApplication.ConsumerApplication_LiveLocationMessage{
			Location: &waConsumerApplication.ConsumerApplication_Location{DegreesLatitude: proto.Float64(1.5), DegreesLongitude: proto.Float64(2.5)},
		}}}
	if got := gometaE2EEText(live); !strings.HasPrefix(got, "[LIVE LOCATION]\n1.500000, 2.500000\n") {
		t.Errorf("live: got %q", got)
	}
}

func TestGometaArmadilloLocationShare(t *testing.T) {
	ecm := &waArmadilloXMA.ExtendedContentMessage{
		TitleText: proto.String("Pinned location"), SubtitleText: proto.String("Main Street"),
		Ctas: []*waArmadilloXMA.ExtendedContentMessage_CTA{{NativeURL: proto.String("messenger://location_share?lat=52.1&long=4.3")}},
	}
	if got, want := gometaArmadilloText(ecm), gometaLocationText(52.1, 4.3, "Pinned location", "Main Street", false); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	link := &waArmadilloXMA.ExtendedContentMessage{MessageText: proto.String("see this"),
		Ctas: []*waArmadilloXMA.ExtendedContentMessage_CTA{{NativeURL: proto.String("https://example.org/p")}}}
	if got := gometaArmadilloText(link); got != "see this\nhttps://example.org/p" {
		t.Errorf("link: got %q", got)
	}
}

func TestGometaGeo(t *testing.T) {
	for in, ok := range map[string]bool{
		"messenger://location_share?lat=52.1&long=4.3": true, "geo:52.1,4.3": true, "52.1,4.3": true,
		"geo:0,0": false, "https://example.org/52.1,4.3": false, "91,4": false, "": false,
	} {
		if _, _, got := gometaGeo(in); got != ok {
			t.Errorf("gometaGeo(%q) ok = %v, want %v", in, got, ok)
		}
	}
}

func TestGometaE2EEStickerIsKeptAsWebP(t *testing.T) {
	sha := []byte{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89}
	m, ok := gometaE2EEMediaOf(gmSticker(t, "/s", "image/webp", sha))
	if !ok || !m.sticker || m.mediaType != whatsmeow.MediaImage || m.extension != ".webp" {
		t.Fatalf("got %+v ok=%v", m, ok)
	}
	if m2, _ := gometaE2EEMediaOf(gmSticker(t, "/s", "", sha)); m2.extension != ".webp" {
		t.Errorf("a sticker with no type: extension %q, want .webp", m2.extension)
	}
}

func TestGometaE2EEStickerIsKeptAsAFile(t *testing.T) {
	h, r := newTestGometa(t)
	sha := make([]byte, 32)
	sha[0] = 0x5a
	r.media["/sticker"] = []byte("RIFFxxxxWEBP")
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "7001", gmSticker(t, "/sticker", "image/webp", sha)))
	a := onlyAttachment(t, r)
	want := filepath.Join(h.purpleUserDir, "gometa", "attachments", "test@example.invalid", hex.EncodeToString(sha)+".webp")
	if a.path != want || a.mimetype != "image/webp" || a.m.id != "7001" || a.m.conv != itoa(gmPeer) {
		t.Errorf("got %+v, want path %q", a, want)
	}
	if data, err := os.ReadFile(want); err != nil || string(data) != "RIFFxxxxWEBP" {
		t.Errorf("file: %q %v", data, err)
	}
}

func TestGometaE2EEPictureKeepsItsCaption(t *testing.T) {
	h, r := newTestGometa(t)
	sha := make([]byte, 32)
	r.media["/img"] = []byte{0xff, 0xd8}
	h.handleE2EEMessage(gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "7002", gmImage(t, "/img", sha, "at the beach")))
	a := onlyAttachment(t, r)
	if !strings.HasSuffix(a.path, ".jpg") || a.m.text != "at the beach" {
		t.Errorf("got %+v", a)
	}
}

func TestGometaE2EEArmadilloLocationIsShown(t *testing.T) {
	h, r := newTestGometa(t)
	evt := gmFBMessage(gmJID(gmPeer), gmJID(gmPeer), "7003", nil)
	evt.Message = &waArmadilloApplication.Armadillo{Payload: &waArmadilloApplication.Armadillo_Payload{
		Payload: &waArmadilloApplication.Armadillo_Payload_Content{Content: &waArmadilloApplication.Armadillo_Content{
			Content: &waArmadilloApplication.Armadillo_Content_ExtendedContentMessage{ExtendedContentMessage: &waArmadilloXMA.ExtendedContentMessage{
				Ctas: []*waArmadilloXMA.ExtendedContentMessage_CTA{{NativeURL: proto.String("messenger://location_share?lat=10&long=20")}},
			}},
		}},
	}}
	h.handleE2EEMessage(evt)
	if m := onlyMessage(t, r); !strings.Contains(m.text, "\n10.000000, 20.000000\n") || m.id != "7003" {
		t.Errorf("got %+v", m)
	}
}

// ---- plain chats ----

func gmInsert(id, text string, sender int64) *table.LSInsertMessage {
	return &table.LSInsertMessage{ThreadKey: gmPeer, SenderId: sender, Text: text, MessageId: id, TimestampMs: 1700000000000}
}

func TestGometaPlainStickerWithoutTextIsShown(t *testing.T) {
	h, r := newTestGometa(t)
	r.media["https://cdn/s.webp"] = []byte("RIFF")
	h.parseTable(&table.LSTable{
		LSInsertMessage:           []*table.LSInsertMessage{gmInsert("mid.1", "", gmPeer)},
		LSInsertStickerAttachment: []*table.LSInsertStickerAttachment{{MessageId: "mid.1", AttachmentFbid: "55", PreviewUrl: "https://cdn/s.webp", PreviewUrlMimeType: "image/webp"}},
	})
	a := onlyAttachment(t, r)
	if filepath.Dir(a.path) != h.attachmentDir() || !strings.HasSuffix(a.path, ".webp") || a.m.id != "mid.1" {
		t.Errorf("got %+v", a)
	}
}

func TestGometaPlainPictureCarriesItsCaption(t *testing.T) {
	h, r := newTestGometa(t)
	r.media["https://cdn/p.jpg"] = []byte{0xff, 0xd8}
	h.parseTable(&table.LSTable{
		LSInsertMessage:        []*table.LSInsertMessage{gmInsert("mid.2", "hello", gmPeer)},
		LSInsertBlobAttachment: []*table.LSInsertBlobAttachment{{MessageId: "mid.2", AttachmentFbid: "66", PreviewUrl: "https://cdn/p.jpg", PreviewUrlMimeType: "image/jpeg"}},
	})
	a := onlyAttachment(t, r)
	if !strings.HasSuffix(a.path, ".jpg") || a.m.text != "hello" {
		t.Errorf("got %+v", a)
	}
	// The same attachment again (the next login's inbox) is not fetched again.
	r.attachments = nil
	h.parseTable(&table.LSTable{
		LSInsertMessage:        []*table.LSInsertMessage{gmInsert("mid.2", "hello", gmPeer)},
		LSInsertBlobAttachment: []*table.LSInsertBlobAttachment{{MessageId: "mid.2", AttachmentFbid: "66", PreviewUrl: "https://cdn/p.jpg", PreviewUrlMimeType: "image/jpeg"}},
	})
	if len(r.fetched) != 1 {
		t.Errorf("fetched %d times, want 1", len(r.fetched))
	}
}

func TestGometaPlainFileKeepsItsName(t *testing.T) {
	h, r := newTestGometa(t)
	r.media["https://cdn/f"] = []byte("%PDF")
	h.parseTable(&table.LSTable{
		LSInsertMessage:        []*table.LSInsertMessage{gmInsert("mid.3", "", gmPeer)},
		LSInsertBlobAttachment: []*table.LSInsertBlobAttachment{{MessageId: "mid.3", AttachmentFbid: "77", PlayableUrl: "https://cdn/f", AttachmentMimeType: "application/pdf", Filename: "Q3 Report.pdf"}},
	})
	if a := onlyAttachment(t, r); !strings.HasSuffix(a.path, "/Q3 Report.pdf") || a.mimetype != "application/pdf" {
		t.Errorf("got %+v", a)
	}
}

func TestGometaPlainFailedDownloadStillShowsTheText(t *testing.T) {
	h, r := newTestGometa(t)
	r.failNext = errors.New("HTTP 403")
	h.parseTable(&table.LSTable{
		LSInsertMessage:        []*table.LSInsertMessage{gmInsert("mid.4", "caption", gmPeer)},
		LSInsertBlobAttachment: []*table.LSInsertBlobAttachment{{MessageId: "mid.4", AttachmentFbid: "88", PreviewUrl: "https://cdn/x.jpg"}},
	})
	if m := onlyMessage(t, r); m.text != "caption" {
		t.Errorf("got %q", m.text)
	}
}

func TestGometaPlainLocationIsShownAsText(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{
		LSInsertMessage:       []*table.LSInsertMessage{gmInsert("mid.5", "", gmPeer)},
		LSInsertXmaAttachment: []*table.LSInsertXmaAttachment{{MessageId: "mid.5", AttachmentFbid: "99", TitleText: "Live location", SubtitleText: "Here"}},
		LSInsertAttachmentCta: []*table.LSInsertAttachmentCta{{AttachmentFbid: "99", Type_: "xma_live_location_sharing", NativeUrl: "52.5,13.4"}},
	})
	m := onlyMessage(t, r)
	if want := gometaLocationText(52.5, 13.4, "Live location", "Here", true); m.text != want {
		t.Errorf("got %q, want %q", m.text, want)
	}
	if len(r.fetched) != 0 {
		t.Errorf("a location fetched %v", r.fetched)
	}
}

func TestGometaPlainThumbsUpIsAnEmoji(t *testing.T) {
	h, r := newTestGometa(t)
	msg := gmInsert("mid.6", "", gmPeer)
	msg.StickerId = 369239263222822
	h.parseTable(&table.LSTable{
		LSInsertMessage:           []*table.LSInsertMessage{msg},
		LSInsertStickerAttachment: []*table.LSInsertStickerAttachment{{MessageId: "mid.6", AttachmentFbid: "1", PreviewUrl: "https://cdn/like.png"}},
	})
	if m := onlyMessage(t, r); m.text != "\U0001F44D" || len(r.fetched) != 0 {
		t.Errorf("got %q, fetched %v", m.text, r.fetched)
	}
}

func TestGometaPlainTextAloneIsUnchanged(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{LSInsertMessage: []*table.LSInsertMessage{gmInsert("mid.7", "hi", gmPeer), gmInsert("mid.8", "", gmPeer)}})
	if m := onlyMessage(t, r); m.text != "hi" || m.id != "mid.7" {
		t.Errorf("got %+v", m)
	}
}
