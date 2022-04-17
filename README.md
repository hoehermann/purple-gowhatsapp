# purple-whatsmeow

A libpurple/Pidgin plugin for WhatsApp. Being developed on Ubuntu 20.04. 

This is a re-write of [purple-gowhatsapp](https://github.com/hoehermann/purple-gowhatsapp/tree/gowhatsapp), switching back-ends from [go-whatsapp](https://github.com/Rhymen/go-whatsapp) to [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow is written by Tulir Asokan. It has multi-device support.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")

### Features

Standard features:

* Connecting to existing account via QR-code.
* Receiving messages, sending messages.
* Receiving files (images, videos, voice, document, stickers).
* Received images are displayed in the conversation window (optional).
* Sending images as image messages.
* Sending ogg audio files as voice messages.
* Sending mp4 video files as video messages.
* Sending other files as documents.
* Fetching all contacts from account, showing friendly names in buddy list, downloading profile pictures ([Markus "nihilus" Gothe](https://github.com/nihilus) for [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy)).
* Sending receipts (configurable).
* Support for socks5 proxies.
* Reasonable support for group chats by [yourealwaysbe](https://github.com/yourealwaysbe).
* Under the hood: Reasonable callback mechanism thanks to [Eion Robb](https://github.com/EionRobb).

Major differences from the go-whatsapp vesion:

* Messages are sent asynchronously.
* Incoming messages are not filtered (formerly by Daniele Rogora, now whatsmeow keeps track of already received messages internally).
* Note: Under the hood, gowhatsapp and whatsmeow use completely different prototocls. For this reason, one must establish a new session (scan QR-code) when switching. All old (non-multi-device) sessions will be invalidated. This is a technical requirement.
* Note: This is not a perfect drop-in replacement for the plug-in with the gowhatsapp back-end. For this reason, it has a different ID: `prpl-hehoe-whatsmeow`

Other improvements:

* Contact presence is regarded (buddies are online and offline).
* Typing notifications are handled.
* Logging happens via purple.
* Messages which only consist of a single URL may be sent as media messages (disabled by default).
* There is an "away" state.
  * For compatibility with the auto-responder plug-in.
  * Other devices (i.e. the main phone) display notifications while plug-in connection is "away".
  * WhatsApp does not send contact presence updates while being "away".
  * Caveat emptor: Other side-effects may occur while using "away" state.

Known issues:

* Contacts:
  * If someone adds you to their contacts and sends you the very first message, the message will not be received. WhatsApp Web shows a notice "message has been delayed – check your phone". This notice is not shown by the plug-in.
* Group Chats:
  * Attachments are downloaded, but link is not shown in group conversation window (not a Purple limitation, tdlib can do it).
  * Cannot send files of any kind to groups (Purple limitation? tdlib can embed images).
  * No notification when being added to a group (the chat will be entered upon receiving a message).

Other planned features:

* Display receipts in conversation window.
* Join group chat via link.
* Option to log out explicitly.
* After download succeeds, write link to chat (for bitlbee).
* Embed stickers into conversation (maybe, WebP support in GDK would be ideal).

These features will not be worked on:

* Accessing microphone and camera for recording voice or video messages.  
  To prepare a voice message, you can use other tools for recording. I like to use [ffmpeg](https://ffmpeg.org/download.html):
  
      ffmpeg -f pulse -i default -ac 1 -ar 16000 -c:a libopus -y voicemessage.ogg # on Linux with PulseAudio

### Building

#### Pre-Built Binaries

* [Nightly Build](https://buildbot.hehoe.de/purple-whatsmeow/builds/) (Windows).

#### Instructions

Dependencies: 

* pidgin (libpurple glib gtk)
* pkg-config
* cmake (3.8 or later)
* make
* go (1.17 or later)
* gcc (6.3.0 or later)

This project uses CMake.

    mkdir build
    cd build
    cmake ..
    cmake --build .
    sudo make install/strip

#### Windows Specific

CMake will try to set-up a development environment automatically. 

Additional dependencies:

* [go 1.17 or newer (32 bit)](https://go.dev/dl/go1.17.5.windows-386.msi)
* [gcc (32 bit)](https://osdn.net/projects/mingw/)

go and gcc must be in `%PATH%`.

MSYS make and CMake generator "MSYS Makefiles" are recommended.  
The project can be opened using Microsoft Visual Studio 2022.  
Compiling with MSVC results in an unusable binary. NOT recommended.  

### Installation

* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

#### Set-Up

* Create a new account  
  You can enter an arbitrary username. 
  It is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, you *recognize yourself in group chats* and Pidgin's logs look sane.  
  Note: Spectrum user reports say, prepending a plus sign works better: `+49123456789@s.whatsapp.net`

* Upon login, a QR code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with WhatsApp Web.

#### Purple Settings

* `qrcode-size`  
  The size of the QR code shown for login purposes, in pixels (default: 256).
  
* `plain-text-login`  
  If set to true (default: false), QR code will be delivered as a text message (for text-only clients).
  
* `fetch-contacts`  
  If set to true (default), buddy list will be populated with contacts sent by server. 
  This is useful for the first login in particular. If enabled while connecting, it will also fetch the current list of WhatsApp groups.
  
* `fake-online`  
  If set to true (default), contacts currently not online will be regarded as "away" (so they still appear in the buddy list).
  If set to false, offline contacts will be regarded as "offline" (no messages can be sent).

* `send-receipt`  
  Selects when to send receipts "double blue tick" notifications:
    * `immediately`: immediately upon message receival
    * `on-interact`: as the user interacts with the conversation window (currently buggy)
    * `on-answer`: as soon as the user sends an answer (default)
    * `never`: never
    
* `inline-images`:
  If set to true (default), images will automatically be downloaded and embedded in the conversation window.
  
* `get-icons`  
  If set to true (default: false), profile pictures are updated every time the plug-in connects.
  
* `bridge-compatibility`  
  Special compatibility setting for protocol bridges like Spectrum or bitlbee. Setting this to true (default: false) will treat system messages just like normal messages, allowing them to be logged and forwarded. This only affects soft errors regarding a specific conversation, e.g. "message could not be sent".
    
* `echo-sent-messages`  
  Selects when to put an outgoing message into the local conversation window:
    * `on-success`: After the WhatsApp *server* has received the message (default). Note: This does not indicate whether the message has been received by the *contact*.
    * `immediately`: Immediately after hitting send (message may not actually have been sent).
    * `never`: Never (some protocol bridges want this).

* `autojoin-chats`  
  Automatically join all chats representing the WhatsApp groups after connecting and every time group information is provided. This is useful for protocol bridges.
  
* `database-address`  
  whatsmeow stores all session information in a SQL database.
  
  This setting can have place-holders:
  
  * `$purple_user_dir` – will be replaced by the user directory, e.g. `~/.purple`.
  * `$username` – will be replaced by the username as entered in the account details.
  
  default: `file:$purple_user_dir/whatsmeow.db?_foreign_keys=on&_busy_timeout=3000`  
  Folder must exist, `whatsmeow.db` is created automatically.
  
  By default, the driver will be `sqlite3` for a file-backed SQLite database. This is not recommended for multi-account-applications (e.g. spectrum or bitlbee) due to a [limitation in the driver](https://github.com/mattn/go-sqlite3/issues/209). The file-system (see addess option) must support locking and be responsive. Network shares (especially SMB) **do not work**.
  
  If the setting starts with `postgres:`, the suffix will be passed to [database/sql.Open](https://pkg.go.dev/database/sql#Open) as `dataSourceName` for the [pq](https://github.com/lib/pq) PostgreSQL driver. At time of writing, there are no further drivers [supported by whatsmeow](https://github.com/tulir/whatsmeow/blob/b078a9e/store/sqlstore/container.go#L34). Support for MySQL/MariaDB has been [requested](https://github.com/tulir/whatsmeow/pull/48). 

* `embed-max-file-size`  
  When set to a value greater than 0 (default, in megabytes), the plug-in tries to detect link-only messages such as `https://example.com/voicemessage.oga` for forwarding.
  
  If enabled, this plug-in tries to download and forward the linked file, choosing the appropriate media type automatically. This way, your contacts do not see a link to an image, video or a voice message, but instead can play the content directly in their app. The message must consist of one URL exactly, including whitespace. For this reason, this mode is incompatbile with Pidgin's OTR plug-in, see [this bug report](https://developer.pidgin.im/ticket/10280). 
  
  At time of writing, the maximum file-size supported by WhatsApp is 16 MB. Further conditions need to be met, see below. 

### Notes

#### Conditions for Sending Media Messages

WhatsApp is very picky about media messages. This is actually a good thing for ensuring compatibility on all devices and clients (Android, iOS, all browsers for WhatsApp Web…).

An image may be sent as an image message (JPEG, `image/jpeg`). This is relatively straight-forward.

A voice message must meet these criteria:

* Mime-Type: `application/ogg`, `audio/ogg`, sent as `audio/ogg; codecs=opus`
* Container: `ogg`
* Codec: `opus`
* Channels: 1 (mono)
* Sample-Rate: 16 kHz

A video message must meet these criteria:

* Mime-Type: `video/mp4`
* Container Major Brand: `mp42` (observed), `isom` (also accepted)
* Moov Atom Location: Beginning (recommended)
* Video Track:
    * Codec: `h264`
    * Pixel Format: `yuv420p` (assumed, not checked)
* Audio Track (optional):
    * Codec: `aac` (not checked)

Not all of these values are checked by the plug-in. Some of these criteria are guessed and may not actually be WhatsApp restrictions.

#### Attachment Handling and Memory Consumption

Attachments (images, videos, voice messages, stickers, document) are *always* downloaded as *soon as the message is processed*. The user is then asked where they want the file to be written. During this time, the file data is residing in memory multiple times:

* in the input buffer
* in the decryption buffer
* in the go → C message buffer
* in the output buffer

On systems with many concurrent connections, this could exhaust memory.

As of writing, whatsmeow does not offer an interface to read the file in chunks.

#### Proxy Support

[whatsmeow](https://github.com/tulir/whatsmeow/blob/9f73bc00d158688a14d0147a93b6b25373facbb8/client.go#L206) offers support for SOCKS5 proxies only. Even if no proxy settings are set in purple, the underlying Go runtime might pick up the `https_proxy` environment variable anyway. 

#### Acknowledgements

* [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy) for initiating the re-write
* [yourealwaysbe](https://github.com/yourealwaysbe) for proper group chats, support and tests against [bitlee](https://github.com/bitlbee/bitlbee)
* [vitalyster](https://github.com/vitalyster) for support, packaging and adjustments for [spectrum2](https://github.com/SpectrumIM/spectrum2)
* Martin Sebald from [https://jabber.hot-chilli.net/](hot-chilli.net) for extensive stress-testing 
* [JimB](https://stackoverflow.com/users/32880/jimb) for golang insights
* [Eion Robb](https://github.com/EionRobb/) for sharing his invaluable purple advice
* [HVV](https://www.hvv.de/) for providing free wifi at their stations

