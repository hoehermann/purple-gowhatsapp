//go:build gometa

package main

// What a received Facebook message shows besides plain text: pictures, files, stickers, locations
// and link previews, in encrypted chats (the whatsmeow client) and plain ones (messagix tables).
// A file is downloaded when it comes and kept under the purple user directory
// (gometa_attachment.go); a location is shown as the WhatsApp protocol shows one (format_location).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waArmadilloXMA"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
)

// ---- encrypted chats ----

// gometaE2EEText is the text an encrypted message's content shows, "" for one with none (media,
// reactions, edits). An ExtendedTextMessage is a text with a link preview: its title and
// description go above the text, as WhatsApp's link previews do (handle_message.go).
func gometaE2EEText(content *waConsumerApplication.ConsumerApplication_Content) string {
	if mt := content.GetMessageText(); mt != nil {
		return mt.GetText()
	}
	if etm := content.GetExtendedTextMessage(); etm != nil {
		var lines []string
		for _, s := range []string{etm.GetTitle(), etm.GetDescription(), etm.GetText().GetText()} {
			if s = strings.TrimSpace(s); s != "" {
				lines = append(lines, s)
			}
		}
		return strings.Join(lines, "\n")
	}
	if lm := content.GetLocationMessage(); lm != nil {
		loc := lm.GetLocation()
		return gometaLocationText(loc.GetDegreesLatitude(), loc.GetDegreesLongitude(), loc.GetName(), lm.GetAddress(), false)
	}
	if llm := content.GetLiveLocationMessage(); llm != nil {
		loc := llm.GetLocation()
		return gometaLocationText(loc.GetDegreesLatitude(), loc.GetDegreesLongitude(), "", "", true)
	}
	return ""
}

// gometaArmadilloText is the text of an encrypted "extended content" message (Armadillo XMA): a
// shared location (a "messenger://location_share?lat=..&long=.." call to action, which is how
// mautrix-meta's msgconv reads one) or a shared link.
func gometaArmadilloText(ecm *waArmadilloXMA.ExtendedContentMessage) string {
	if ecm == nil {
		return ""
	}
	body := strings.TrimSpace(ecm.GetMessageText())
	for _, cta := range ecm.GetCtas() {
		native := cta.GetNativeURL()
		if lat, lon, ok := gometaGeo(native); ok {
			return gometaLocationText(lat, lon, ecm.GetTitleText(), ecm.GetSubtitleText(), false)
		}
		if strings.HasPrefix(native, "https://") && !strings.Contains(body, native) {
			if body != "" {
				body += "\n"
			}
			body += native
		}
	}
	if body == "" {
		body = strings.TrimSpace(ecm.GetTitleText())
	}
	return body
}

// gometaGeo reads a position from what Facebook names one with: "messenger://location_share?
// lat=..&long=..", "geo:lat,lng", or a bare "lat,lng" (a live location's native URL, which
// mautrix-meta turns into "geo:" + it).
func gometaGeo(s string) (lat, lon float64, ok bool) {
	s = strings.TrimSpace(s)
	if u, err := url.Parse(s); err == nil && u.Scheme == "messenger" && u.Host == "location_share" {
		return gometaLatLon(u.Query().Get("lat"), u.Query().Get("long"))
	}
	s = strings.TrimPrefix(s, "geo:")
	if i := strings.IndexAny(s, ";?"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	return gometaLatLon(parts[0], parts[1])
}

func gometaLatLon(a, b string) (float64, float64, bool) {
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 || (lat == 0 && lon == 0) {
		return 0, 0, false
	}
	return lat, lon, true
}

// gometaE2EEMedia is an encrypted message's media, as far as it can be known before download.
type gometaE2EEMedia struct {
	transport *waMediaTransport.WAMediaTransport
	mediaType whatsmeow.MediaType
	mimetype  string
	filename  string // a document's own name, without its extension
	extension string
	caption   string
	sticker   bool
}

// gometaE2EEMediaOf finds the media in an encrypted message's content: a picture, video, voice
// message, file or sticker. A sticker is fetched as an image, as mautrix-meta does, and kept as
// .webp like a received WhatsApp sticker (sticker_file_type), unless it says it is something else.
func gometaE2EEMediaOf(content *waConsumerApplication.ConsumerApplication_Content) (gometaE2EEMedia, bool) {
	var m gometaE2EEMedia
	if vm := content.GetVideoMessage(); vm != nil {
		t, err := vm.Decode()
		if err != nil {
			return m, false
		}
		m.transport, m.mediaType = t.GetIntegral().GetTransport(), whatsmeow.MediaVideo
		m.caption = vm.GetCaption().GetText()
		if m.caption == "" {
			m.caption = t.GetAncillary().GetCaption().GetText()
		}
	} else if im := content.GetImageMessage(); im != nil {
		t, err := im.Decode()
		if err != nil {
			return m, false
		}
		m.transport, m.mediaType = t.GetIntegral().GetTransport(), whatsmeow.MediaImage
		m.caption = im.GetCaption().GetText()
	} else if am := content.GetAudioMessage(); am != nil {
		t, err := am.Decode()
		if err != nil {
			return m, false
		}
		m.transport, m.mediaType = t.GetIntegral().GetTransport(), whatsmeow.MediaAudio
	} else if dm := content.GetDocumentMessage(); dm != nil {
		t, err := dm.Decode()
		if err != nil {
			return m, false
		}
		m.transport, m.mediaType = t.GetIntegral().GetTransport(), whatsmeow.MediaDocument
		m.filename = dm.GetFileName()
	} else if sm := content.GetStickerMessage(); sm != nil {
		t, err := sm.Decode()
		if err != nil {
			return m, false
		}
		m.transport, m.mediaType, m.sticker = t.GetIntegral().GetTransport(), whatsmeow.MediaImage, true
	} else {
		return m, false
	}
	if m.transport.GetIntegral() == nil {
		return m, false
	}
	m.mimetype = m.transport.GetAncillary().GetMimetype()
	if m.filename != "" { // a document keeps its real name, shown on the chip
		m.extension = filepath.Ext(m.filename)
		m.filename = strings.TrimSuffix(m.filename, m.extension)
	}
	if m.sticker {
		m.extension = gometaStickerExtension(m.mimetype)
	} else if m.extension == "" {
		m.extension = extension_from_mimetype(&m.mimetype)
	}
	return m, true
}

// gometaStickerExtension: a sticker is a WebP unless its type says otherwise.
func gometaStickerExtension(mimetype string) string {
	if mimetype == "" || mimetype == "image/webp" {
		return ".webp"
	}
	return extension_from_mimetype(&mimetype)
}

// fileName is the name this media is kept under: its content hash, a document's own name in a
// folder of that name.
func (m gometaE2EEMedia) fileName() string {
	return gometaFileName(hex.EncodeToString(m.transport.GetIntegral().GetFileSHA256()), m.filename, m.extension)
}

// ---- plain chats ----

// gometaFBAttachment is one attachment of a plain chat's message: a file to fetch from url, or
// (url "") a text to show, for a shared location or link.
type gometaFBAttachment struct {
	url, mime, filename, fbid string
	sticker                   bool
	text                      string
}

// gometaPlainAttachments gathers a table's attachments by the message they belong to, as
// table.WrapMessages does. WrapMessages itself is not used: it returns an upserted message only
// when the table also has its thread's message range, and parseTable shows every upsert.
func gometaPlainAttachments(tbl *table.LSTable) map[string][]gometaFBAttachment {
	out := make(map[string][]gometaFBAttachment)
	seen := make(map[string]bool)
	add := func(msgID string, a gometaFBAttachment) {
		if msgID == "" || (a.url == "" && a.text == "") {
			return
		}
		// Facebook sometimes sends the same blob row twice (mautrix-meta dedupes by fbid too).
		if a.fbid != "" {
			if seen[msgID+"\x00"+a.fbid] {
				return
			}
			seen[msgID+"\x00"+a.fbid] = true
		}
		out[msgID] = append(out[msgID], a)
	}
	media := func(playable, playableMime, attMime, preview, previewMime string) (string, string) {
		if playable != "" {
			if playableMime == "" {
				playableMime = attMime
			}
			return playable, playableMime
		}
		return preview, previewMime
	}
	for _, b := range tbl.LSInsertBlobAttachment {
		u, mt := media(b.PlayableUrl, b.PlayableUrlMimeType, b.AttachmentMimeType, b.PreviewUrl, b.PreviewUrlMimeType)
		add(b.MessageId, gometaFBAttachment{url: u, mime: mt, filename: b.Filename, fbid: b.AttachmentFbid})
	}
	for _, a := range tbl.LSInsertAttachment {
		u, mt := media(a.PlayableUrl, a.PlayableUrlMimeType, a.AttachmentMimeType, a.PreviewUrl, a.PreviewUrlMimeType)
		add(a.MessageId, gometaFBAttachment{url: u, mime: mt, filename: a.Filename, fbid: a.AttachmentFbid})
	}
	for _, s := range tbl.LSInsertStickerAttachment {
		u, mt := media(s.PlayableUrl, s.PlayableUrlMimeType, "", s.PreviewUrl, s.PreviewUrlMimeType)
		add(s.MessageId, gometaFBAttachment{url: u, mime: mt, fbid: s.AttachmentFbid, sticker: true})
	}
	ctas := make(map[string]*table.LSInsertAttachmentCta, len(tbl.LSInsertAttachmentCta))
	for _, c := range tbl.LSInsertAttachmentCta {
		ctas[c.AttachmentFbid] = c
	}
	for _, x := range tbl.LSInsertXmaAttachment {
		add(x.MessageId, gometaXMAAttachment(x, ctas[x.AttachmentFbid]))
	}
	return out
}

// gometaXMAAttachment reads an "extended" attachment: a location when its call to action names a
// position (a live location's native URL is "lat,lng", as mautrix-meta reads it), else a shared
// link or post, shown as its title and link.
func gometaXMAAttachment(x *table.LSInsertXmaAttachment, cta *table.LSInsertAttachmentCta) gometaFBAttachment {
	a := gometaFBAttachment{fbid: x.AttachmentFbid}
	if cta != nil {
		for _, u := range []string{cta.NativeUrl, cta.ActionUrl} {
			if lat, lon, ok := gometaGeo(u); ok {
				a.text = gometaLocationText(lat, lon, x.TitleText, x.SubtitleText, strings.Contains(cta.Type_, "live_location"))
				return a
			}
		}
	}
	link := x.ActionUrl
	if cta != nil && cta.ActionUrl != "" {
		link = cta.ActionUrl
	}
	link = gometaRemoveLPHP(link)
	var lines []string
	for _, s := range []string{x.TitleText, link} {
		if s = strings.TrimSpace(s); s != "" {
			lines = append(lines, s)
		}
	}
	a.text = strings.Join(lines, "\n")
	return a
}

// gometaRemoveLPHP unwraps Facebook's link redirector (l.php?u=...), as mautrix-meta does.
func gometaRemoveLPHP(addr string) string {
	if u, err := url.Parse(addr); err == nil && u.Path == "/l.php" && u.Query().Get("u") != "" {
		return u.Query().Get("u")
	}
	return addr
}

// gometaThumbsUpStickers are Facebook's like button, which mautrix-meta shows as a 👍.
var gometaThumbsUpStickers = map[int64]bool{369239263222822: true, 369239343222814: true, 369239383222810: true}

// fileName is the name a plain chat's attachment is kept under: a hash of its fbid (or URL), so the
// same attachment arriving again (each login's inbox table repeats the latest messages) is not
// fetched twice; a document's own name in a folder of that name.
func (a gometaFBAttachment) fileName() string {
	key := a.fbid
	if key == "" {
		key = a.url
	}
	sum := sha256.Sum256([]byte(key))
	var ext, name string
	if a.sticker {
		ext = gometaStickerExtension(a.mime)
	} else if a.filename != "" {
		ext = filepath.Ext(a.filename)
		name = strings.TrimSuffix(a.filename, ext)
	}
	if ext == "" {
		mt := a.mime
		ext = extension_from_mimetype(&mt)
	}
	return gometaFileName(hex.EncodeToString(sum[:]), name, ext)
}

// gometaPlainBody is what a plain chat's message shows: the text of its location or link
// attachments with the message's own text, and the attachments that are files to fetch.
func gometaPlainBody(text string, stickerID int64, atts []gometaFBAttachment) (string, []gometaFBAttachment) {
	var files []gometaFBAttachment
	var lines []string
	if gometaThumbsUpStickers[stickerID] && len(atts) == 1 && atts[0].sticker && text == "" {
		return "\U0001F44D", nil
	}
	for _, a := range atts {
		if a.url != "" {
			files = append(files, a)
		} else if a.text != "" && !strings.Contains(text, a.text) {
			lines = append(lines, a.text)
		}
	}
	if text != "" {
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n"), files
}

// gometaLocationText describes a shared location as the WhatsApp protocol does (format_location).
func gometaLocationText(lat, lon float64, name, address string, live bool) string {
	text := "[LOCATION]"
	if live {
		text = "[LIVE LOCATION]"
	}
	if name = strings.TrimSpace(name); name != "" {
		text += " " + name
	}
	if address = strings.TrimSpace(address); address != "" {
		text += "\n" + address
	}
	return text + fmt.Sprintf("\n%f, %f\nhttps://www.openstreetmap.org/?mlat=%f&mlon=%f", lat, lon, lat, lon)
}

// gometaE2EEContact is the text of a received contact card (or several), as the WhatsApp protocol
// shows one (format_contact), and whether the content is one.
func gometaE2EEContact(content *waConsumerApplication.ConsumerApplication_Content) (string, bool) {
	text := func(cm *waConsumerApplication.ConsumerApplication_ContactMessage) string {
		dec, err := cm.Decode()
		if err != nil {
			return ""
		}
		return format_contact(dec.GetAncillary().GetDisplayName(), dec.GetIntegral().GetVcard())
	}
	if cm := content.GetContactMessage(); cm != nil {
		return text(cm), true
	}
	if ca := content.GetContactsArrayMessage(); ca != nil {
		var cards []string
		for _, cm := range ca.GetContacts() {
			if t := text(cm); t != "" {
				cards = append(cards, t)
			}
		}
		return strings.Join(cards, "\n"), true
	}
	return "", false
}
