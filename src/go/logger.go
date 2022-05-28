package main

import (
	"fmt"
	waLog "go.mau.fi/whatsmeow/util/log"
	"strings"
)

type purpleLogger struct {
	account *PurpleAccount
	topic   string
}

func (l *purpleLogger) formatf(msg string, args ...interface{}) string {
	return fmt.Sprintf("[%s] %s\n", l.topic, fmt.Sprintf(msg, args...))
}

// numeric error level values corresponding to PurpleDebugLevel defined in libpurple/debug.h.

func (l *purpleLogger) Debugf(msg string, args ...interface{}) {
	purple_debug(1, l.formatf(msg, args...))
}
func (l *purpleLogger) Infof(msg string, args ...interface{}) {
	purple_debug(2, l.formatf(msg, args...))
}
func (l *purpleLogger) Warnf(msg string, args ...interface{}) {
	purple_debug(3, l.formatf(msg, args...))
}
func (l *purpleLogger) Errorf(msg string, args ...interface{}) {
	// an error is an error. do not log, but forward error to purple.
	// this is useful for exposing low-level problems to the user
	errorString := l.formatf(msg, args...)
	errorLevel := ERROR_FATAL
	if strings.Contains(errorString, "Error reading from websocket") {
		// this is triggered by network error, it is non-fatal
		// checking for "connection reset by peer" or "close 1006" would be more appropriate, but these messages are locale dependent
		errorLevel = ERROR_TRANSIENT
	}
	purple_error(l.account, errorString, errorLevel)
}

func (l *purpleLogger) Sub(topic string) waLog.Logger {
	return &purpleLogger{account: l.account, topic: fmt.Sprintf("%s/%s", l.topic, topic)}
}

func PurpleLogger(account *PurpleAccount, topic string) waLog.Logger {
	return &purpleLogger{account: account, topic: topic}
}
