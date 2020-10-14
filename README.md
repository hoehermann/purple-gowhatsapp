# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp **Web**.

Powered by [go-whatsapp](https://github.com/Rhymen/go-whatsapp), which is written by Lucas Engelke.

Being developed on Ubuntu 18.04.  
Last seen working with [go-whatsapp 9b4bc38](https://github.com/Rhymen/go-whatsapp/commit/9b4bc38).

### Building

* Build using the supplied Makefile.  
  Optional: Try `make update-dep` for the most recent go-whatsapp version.

### Pre-Built Binaries

* Download a [nightly build](https://buildbot.hehoe.de/purple-gowhatsapp/builds/) (Ubuntu 18.04 and Windows).
* Tested [Windows binaries](https://github.com/hoehermann/purple-gowhatsapp/wiki/Windows-Build) are kindly provided by [EionRobb](https://github.com/EionRobb) on occasion.

### Installation

* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

### Set-up

* Create a new account  
  You can enter an arbitrary username. 
  However, it is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, Pidgin's logs look sane.
* Upon login, a QR code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with the browser-based WhatsApp Web.  
  Alternatively, you can receive the login code as an image message, as a file or as plain text (user configurable).
* Note: Some settings only take effect after a re-connect.

Please also notice the wiki page regarding [common problems](https://github.com/hoehermann/purple-gowhatsapp/wiki/Common-Problems).

#### spectrum2 specifics

[Spectrum 2](https://spectrum.im/) users must set this plug-in's option `system-messages-are-ordinary-messages` to true. By default, the log-in message is a system message and Spectrum 2 ignores system messages.

### Features

* Receive text messages.
* Sending text messages.
* Download files from image, audio, media, and document messages.  
  Files are downloaded to `~/.pidgin/gowhatsapp`.
* Under the hood: Reasonable callback mechanism thanks to [Eiron Robb](https://github.com/EionRobb).
* Fetch contacts from phone, keep track of time last seen, download of user profile pictures courtesy of [Markus Gothe](https://github.com/nihilus).

*Note:* You may need to force the TLS version to 1.2 using the NSS plug-in for download of user profile pictures to work on some systems.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")  

### Missing Features

* Anything beyond simple messaging, really

### What could be done next

From approximate most important to approximate least interesting.

* Find memory leaks.
* Support [stickers](https://github.com/Rhymen/go-whatsapp/commit/d7754af4a6b7209d88132b5e498c98f12fb67f70) #32.
* Improve [proxy support](https://github.com/Rhymen/go-whatsapp/blob/master/examples/loginWithProxy/main.go): Get proxy configuration from environment variables, if requested.
* Use a callback for getting current preferences everywhere consistently.
* Have purple handle the message, then conditionally request download where appropriate.
* Support group conversations properly.
* Sort old messages by date.
* Improve spectrum support:
  * Make online status work.
  * Handle incoming files the way purple-skypeweb does.
* Support sending document messages.
* Wait for server message received acknowledgement before displaying sent message locally.
* Do not block while sending message.
* Mark messages as "read" only after the user interacted with the conversation #47.
* Defer logging debug messages from go-whatsapp to `purple_debug_info()`.
* Refactor the file-handling code as it is really ugly.
* Be compatible with the ["Autoreply"](https://github.com/EionRobb/purple-gowhatsapp/issues/3#issuecomment-555814663) plug-in by having an "Away" state.
* Be compatible with the "Conversation Colors" plug-in.
* Consistently use dashes in key names.
