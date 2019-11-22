# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp **Web**.

Powered by [go-whatsapp](https://github.com/Rhymen/go-whatsapp), which is written by Lucas Engelke.

Being developed on Ubuntu 18.04.  
Last seen working with [go-whatsapp d7754af](https://github.com/Rhymen/go-whatsapp/commit/d7754af).

### Building

* Build using the supplied Makefile.  
  Optional: Try `make update-dep` for the most recent go-whatsapp version.

### Pre-Built Binaries

* Download a [nightly build](https://buildbot.hehoe.de/purple-gowhatsapp/builds/) (Ubuntu 18.04).
* Win32 binaries might be obtained from [EionRobb](https://github.com/EionRobb/purple-gowhatsapp).

### Installation

* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

### Set-up

* Create a new account  
  You can enter an arbitrary username. 
  However, it is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, Pidgin's logs look sane.
* Upon login, a fake conversation should pop up, showing a link to a QR code.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with the browser-based WhatsApp Web.
* Note: Some settings only take effect after a re-connect.

Please also notice the wiki page regarding [common problems](Common-Problems).

#### spectrum2 specifics

[Spectrum 2](https://spectrum.im/) users must set this plug-in's option `system-messages-are-ordinary-messages` to true. By default, the log-in message is a system message and Spectrum 2 ignores system messages.

### Features

* Receive text messages.
* Sending text messages.
* Download files from image, audio, media, and document messages.  
  Files are downloaded to `~/.pidgin/gowhatsapp`.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")  

### Missing Features

* Anything beyond simple messaging, really

### What could be done next

From approximate most important to approximate least interesting.

* Display login QR code via the [request API](https://github.com/EionRobb/pidgin-opensteamworks/blob/master/steam-mobile/libsteam.c#L378-L412).
* Support [stickers](https://github.com/Rhymen/go-whatsapp/commit/d7754af4a6b7209d88132b5e498c98f12fb67f70).
* Use a call-back for getting current preferences everywhere consistently.
* If file download failed, do not try to download again (make user configurable).
* If file download is disabled, mention file only once.
* Have purple handle the message, then conditionally request download where appropriate.
* Add option to ignore invalid message IDs silently.
* Sanitize invalid message IDs (e.g. `I/fb36`).
* Support group conversations properly.
* Pin go dependency version.
* Sort old messages by date.
* Improve spectrum support:
  * Make online status work.
  * Handle incoming files the way purple-skypeweb does.
* Detect media file mime type for sending files.
* Support sending image, audio, media, and document messages by drag-and-drop.
* Find out how whatsapp represents newlines.
* Wait for server message received acknowledgement before displaying sent message locally.
* Do not block while sending message.
* Be compatible with the ["Autoreply"](https://github.com/EionRobb/purple-gowhatsapp/issues/3#issuecomment-555814663) plug-in.
* Find spurious segfault.
* Consistently use dashes in key names.
