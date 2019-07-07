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

.PHONY:	all FAILNOPURPLE clean gdb install

LOCALES = $(patsubst %.po, %.mo, $(wildcard po/*.po))

all: $(TARGET)

purplegwa.a: purplegwa.go purplegwa-media.go
	$(GO) get github.com/Rhymen/go-whatsapp
	$(GO) get github.com/skip2/go-qrcode
	$(GO) build -buildmode=c-archive -o purplegwa.a purplegwa.go purplegwa-media.go

$(TARGET): $(PURPLE_C_FILES) $(PURPLE_COMPAT_FILES) purplegwa.a
	$(CC) -fPIC $(CFLAGS) $(CPPFLAGS) -shared -o $@ $^ $(LDFLAGS) `$(PKG_CONFIG) purple glib-2.0 --libs --cflags` $(INCLUDES) -Ipurple2compat -g -ggdb

FAILNOPURPLE:
	echo "You need libpurple development headers installed to be able to compile this plugin"

clean:
	rm -f $(TARGET) purplegwa.a

install: $(TARGET)
	mkdir -m 0755 -p $(DEST)
	install -m 0755 -p $(TARGET) $(DEST)
	
gdb:
	gdb --args pidgin -d -c .pidgin
