package main

import (
	"fmt"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type purpleLogger struct {
	account *PurpleAccount
	topic   string
}

func (l *purpleLogger) formatf(msg string, args ...interface{}) string {
	return fmt.Sprintf("[%s] %s\n", l.topic, fmt.Sprintf(msg, args...))
}

// numieric level values corresponding to PurpleDebugLevel defined in libpurple/debug.h.

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
	if l.account == nil {
		// error not relatable to a specific account
		// assume problem with database – forward to all accounts
		purple_debug(4, l.formatf(msg, args...))
		purple_error(nil, fmt.Sprintf("Error in component %s. This affects all active connections. View debug log for more information.", l.topic), ERROR_FATAL)
	} else {
		// an error is an error. do not log, but forward error to purple.
		// this is useful for exposing low-level problems to the user
		purple_error(l.account, l.formatf(msg, args...), ERROR_FATAL)
	}
}

func (l *purpleLogger) Sub(topic string) waLog.Logger {
	return &purpleLogger{account: l.account, topic: fmt.Sprintf("%s/%s", l.topic, topic)}
}

func PurpleLogger(account *PurpleAccount, topic string) waLog.Logger {
	return &purpleLogger{account: account, topic: topic}
}
