package main

import (
	"fmt"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type purpleLogger struct {
	topic string
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
	purple_debug(4, l.formatf(msg, args...))
}

func (l *purpleLogger) Sub(topic string) waLog.Logger {
	return &purpleLogger{topic: fmt.Sprintf("%s/%s", l.topic, topic)}
}

func PurpleLogger(topic string) waLog.Logger {
	return &purpleLogger{topic: topic}
}
