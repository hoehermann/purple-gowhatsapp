PROTOC_C ?= protoc-c
PKG_CONFIG ?= pkg-config

# Note: Use "-C .git" to avoid ascending to parent dirs if .git not present
GIT_REVISION_ID = $(shell git -C .git rev-parse --short HEAD 2>/dev/null)
PLUGIN_VERSION ?= $(shell cat VERSION)~git$(GIT_REVISION_ID)

CFLAGS	?= -O2 -g -pipe -Wall
LDFLAGS ?= -Wl,-z,relro

CFLAGS  += -std=c99 -DGOWHATSAPP_PLUGIN_VERSION='"$(PLUGIN_VERSION)"' -DMARKDOWN_PIDGIN

CC ?= gcc
GO ?= go
GIT ?= git

ifeq ($(shell $(PKG_CONFIG) --exists purple 2>/dev/null && echo "true"),)
  TARGET = FAILNOPURPLE
  DEST =
else
  TARGET = libgowhatsapp.so
  DEST = $(DESTDIR)`$(PKG_CONFIG) --variable=plugindir purple`
  LOCALEDIR = $(DESTDIR)$(shell $(PKG_CONFIG) --variable=datadir purple)/locale
endif

CFLAGS += -DLOCALEDIR=\"$(LOCALEDIR)\"

PURPLE_COMPAT_FILES :=
PURPLE_C_FILES := libgowhatsapp.c $(C_FILES)

.PHONY:	all FAILNOPURPLE clean update-dep gdb install

LOCALES = $(patsubst %.po, %.mo, $(wildcard po/*.po))

all: $(TARGET)

GO_WHATSAPP_A = $(shell $(GO) env GOPATH)/pkg/$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)/github.com/Rhymen/go-whatsapp.a
GO_WHATSAPP_GIT = $(shell $(GO) env GOPATH)/src/github.com/Rhymen/go-whatsapp/.git
GO_WHATSAPP_DATE =$(shell $(GIT) --git-dir="$(GO_WHATSAPP_GIT)" log -1 --date=rfc --format=%cd)

update-dep:
	$(GO) get -u github.com/Rhymen/go-whatsapp
	touch -d "$(GO_WHATSAPP_DATE)" $(GO_WHATSAPP_A)

$(GO_WHATSAPP_A):
	$(GO) get github.com/Rhymen/go-whatsapp

gowhatsapp_trigger_read_dummy.o: gowhatsapp_trigger_read_dummy.c
	gcc -o gowhatsapp_trigger_read_dummy.o -c gowhatsapp_trigger_read_dummy.c

purplegwa.a: purplegwa.go purplegwa-media.go $(GO_WHATSAPP_A) gowhatsapp_trigger_read_dummy.o
	$(GO) get github.com/skip2/go-qrcode
	$(GO) build -buildmode=c-archive -o purplegwa.a purplegwa.go purplegwa-media.go

$(TARGET): $(PURPLE_C_FILES) $(PURPLE_COMPAT_FILES) purplegwa.a
	$(CC) -fPIC $(CFLAGS) $(CPPFLAGS) -shared -o $@ $^ $(LDFLAGS) `$(PKG_CONFIG) purple glib-2.0 --libs --cflags` $(INCLUDES) -Ipurple2compat -g -ggdb

FAILNOPURPLE:
	echo "You need libpurple development headers installed to be able to compile this plugin"

clean:
	rm -f $(TARGET) purplegwa.a gowhatsapp_trigger_read_dummy.o

install: $(TARGET)
	mkdir -m 0755 -p $(DEST)
	install -m 0755 -p $(TARGET) $(DEST)
	
gdb:
	gdb --args pidgin -d -c .pidgin
