$(warning This Makefile exists for reference purposes. It is not maintained. Use CMake instead.)

export GO_FLAGS
export CGO_CFLAGS = -DPLUGIN_VERSION=$(shell cat VERSION) $(shell pkg-config --cflags glib-2.0 purple opusfile gdk-pixbuf-2.0)
export CGO_LDFLAGS = $(shell pkg-config --libs glib-2.0 purple opusfile gdk-pixbuf-2.0)

all: libwhatsmeow.so

go.mod: go.mod.in
	cp go.mod.in go.mod

go.sum: go.mod
	go mod tidy

libwhatsmeow.a libwhatsmeow.h: bridge.c bridge.go bridge.h constants.c constants.h go.mod go.sum groups.go handle_message.go handler.go logger.go login.go mark_read.go message_cache.go opusreader.c opusreader.h presence.go profile.go send_file_checks.go send_file.go send_message.go
	go build -buildmode=c-archive -o libwhatsmeow.a $(GO_FLAGS)

libwhatsmeow.so: libwhatsmeow.a libwhatsmeow.h constants.c constants.h glue/blist.c glue/bridge.c glue/commands.c glue/display_message.c glue/gowhatsapp.h glue/groups.c glue/handle_attachment.c glue/init.c glue/login.c glue/options.c glue/pixbuf.c glue/pixbuf.h glue/presence.c glue/process_message.c glue/purple_compat.h glue/qrcode.c glue/receipt.c glue/send_file.c glue/send_message.c
	$(CC) -shared -fPIC -o libwhatsmeow.so constants.c glue/blist.c glue/bridge.c glue/commands.c glue/display_message.c glue/groups.c glue/handle_attachment.c glue/init.c glue/login.c glue/options.c glue/pixbuf.c glue/presence.c glue/process_message.c glue/qrcode.c glue/receipt.c glue/send_file.c glue/send_message.c -I. libwhatsmeow.a $(CGO_CFLAGS) $(CGO_LDFLAGS)

clean:
	rm -f libwhatsmeow.a libwhatsmeow.h
	rm -f libwhatsmeow.so
