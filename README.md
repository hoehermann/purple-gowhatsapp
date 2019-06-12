# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp **Web**.

Powered by [go-whatsapp](https://github.com/Rhymen/go-whatsapp), which is written by Lucas Engelke.

Being developed on Ubuntu 18.04.

### Building

* Have [go-whatsapp cc103e6](https://github.com/Rhymen/go-whatsapp/commit/cc103e60c5a4519207981cb47c011e06c448c80ff).
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

### Features

* Receive text messages
* Receive image messages
* Sending text messages

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")  

### Missing Features

* Anything beyond simple messaging, really

### Known Problems

* Images (including QR code for login) are not displayed, if Pidgin's "Conversation Colors" plug-in is enabled.

### What could be done next

* Pin go dependency version.
* Sort old messages by date.
* Support group conversations properly.
* Implement receiving audio, media, and document messages.
* Wait for server message received acknowledgement before displaying sent message locally.
* Do not block while sending message.
* Find spurious segfault.
* Implement sending image, audio, media, and document messages.
* Consistently use dashes in key names.
