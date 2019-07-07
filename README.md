# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp **Web**.

Powered by [go-whatsapp](https://github.com/Rhymen/go-whatsapp), which is written by Lucas Engelke.

Being developed on Ubuntu 18.04.

### Building

* Have [go-whatsapp 4c0e263](https://github.com/Rhymen/go-whatsapp/commit/4c0e263).
* Build using the supplied Makefile.
* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

### Set-up

* Create a new account  
  You can enter an arbitrary username. 
  However, it is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, Pigin's logs look sane.
* Upon login, a fake conversation should pop up, showing a QR code.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with the browser-based WhatsApp Web.
* Note: Some settings only take effect after a re-connect.

### Features

* Receive text messages
* Sending text messages
* Download files from image, audio, media, and document messages  
  Files are downloaded to `~/.pidgin/gowhatsapp`.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")  

### Missing Features

* Anything beyond simple messaging, really

### What could be done next

* Pin go dependency version.
* Sort old messages by date.
* Support group conversations properly.
* Use a pipe instead of polling.
* Use a call-back for getting current preferences rather than setting them during log-in.
* Reintroduce inline display of images.
* Detect media file mime type.
* Support sending image, audio, media, and document messages by drag-and-drop.
* Find out how whatsapp represents newlines.
* Wait for server message received acknowledgement before displaying sent message locally.
* Do not block while sending message.
* Find spurious segfault.
* Consistently use dashes in key names.
