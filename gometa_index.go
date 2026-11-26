//go:build gometa

package main

// The messages shown recently, by id. Facebook names the message an edit, a deletion or a reaction
// is about only by its id, and an edit or a deletion of a plain chat's message does not even say
// which chat it was in. The index finds that, and the text to quote, as the WhatsApp protocol's
// message cache (message_cache.go) does for WhatsApp. It is kept in memory only, so a change to a
// message from before the plug-in was started is not shown.

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// gometaIndexSize bounds the messages kept.
const gometaIndexSize = 2000

type gometaIndex struct {
	mu    sync.Mutex
	byID  map[string]gometaIncoming
	order []string // oldest first
}

func (x *gometaIndex) add(m gometaIncoming) {
	if m.id == "" {
		return
	}
	m.quotedText, m.quotedFrom = "", ""
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.byID == nil {
		x.byID = make(map[string]gometaIncoming)
	}
	if _, known := x.byID[m.id]; !known {
		x.order = append(x.order, m.id)
		if len(x.order) > gometaIndexSize {
			delete(x.byID, x.order[0])
			x.order = x.order[1:]
		}
	}
	x.byID[m.id] = m
}

func (x *gometaIndex) get(id string) (gometaIncoming, bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	m, ok := x.byID[id]
	return m, ok
}

func (x *gometaIndex) setText(id, text string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if m, ok := x.byID[id]; ok {
		m.text = text
		x.byID[id] = m
	}
}

// editLine is the line an edit adds to the chat of the message shown under id, by its sender, as
// the WhatsApp protocol shows an edit: "[EDIT] " and the new text. ok is false for a message not
// shown here.
func (x *gometaIndex) editLine(id, text string, now int64) (m gometaIncoming, ok bool) {
	m, ok = x.get(id)
	if !ok {
		return m, false
	}
	x.setText(id, text) // a later reaction quotes the new text
	m.text, m.id, m.ts = "[EDIT] "+text, "", now
	return m, true
}

// deleteLine is the line a "delete for everyone" adds to the chat of the message shown under id,
// by its sender, as the WhatsApp protocol shows one. ok is false for a message not shown here.
func (x *gometaIndex) deleteLine(id string, now int64) (m gometaIncoming, ok bool) {
	m, ok = x.get(id)
	if !ok {
		return m, false
	}
	m.text = fmt.Sprintf("deleted %s.", gometaDescribe(m.text, true))
	m.id, m.ts = "", now
	return m, true
}

// reactionLine is the line a reaction adds to its chat, by whoever reacted, as the WhatsApp
// protocol shows one. emoji "" means the reaction was taken back.
func (x *gometaIndex) reactionLine(conv, target, actor, emoji string, now int64) gometaIncoming {
	shown, known := x.get(target)
	quote := gometaDescribe(shown.text, known)
	text := fmt.Sprintf("reacted with %s to %s.", emoji, quote)
	if emoji == "" {
		text = fmt.Sprintf("removed their reaction to %s.", quote)
	}
	return gometaIncoming{conv: conv, who: actor, text: text, ts: now}
}

// gometaDescribe names a message for a reaction or a deletion, as describe_cached_message
// (handle_message.go) does for WhatsApp.
func gometaDescribe(text string, known bool) string {
	if !known {
		return "an earlier message"
	}
	if text == "" {
		return "an attachment"
	}
	ellipsis := ""
	if len([]rune(text)) > 50 {
		ellipsis = "…"
	}
	return fmt.Sprintf("message „%.50s%s“", text, ellipsis)
}

// gometaQuote puts the text a message replies to above it, as the WhatsApp protocol shows a quote.
func gometaQuote(quoted string) string {
	return "> " + strings.Join(strings.Split(quoted, "\n"), "\n> ") + "\n"
}

// gometaParseID reads an fbid or thread key, 0 for anything else.
func gometaParseID(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
