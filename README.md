# purple-whatsmeow

A libpurple/Pidgin plugin for WhatsApp. Being developed on Ubuntu 20.04. 

This is a re-write of [purple-gowhatsapp](https://github.com/hoehermann/purple-gowhatsapp/tree/gowhatsapp), switching back-ends from [go-whatsapp](https://github.com/Rhymen/go-whatsapp) to [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow is written by Tulir Asokan. It has multi-device support.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")

### Features

Standard features:

* Connecting to existing account via QR-code.
* Receiving messages, sending messages.
* Receiving files (images, videos, voice, document, stickers).
* Sending images (everything else is considered a document).
* Fetching all contacts from account, show friendly names in buddy list.
* Sending receipts (configurable).
* Reasonable support for group chats by [yourealwaysbe](https://github.com/yourealwaysbe).
* Under the hood: Reasonable callback mechanism thanks to [Eiron Robb](https://github.com/EionRobb).

Major differences from the go-whatsapp vesion:

* Messages are sent asynchronously.
* Incoming messages are not filtered (whatsmeow keeps track of already received messages internally).
* Note: Under the hood, gowhatsapp and whatsmeow use completely different prototocls. whatsmeow actually uses the same mechanics as Signal. For this reason, one must establish a new session (scan QR-code) when switching. All sessions will be invalidated.

Other improvements:

* Contact presence is regarded (buddies are online and offline).
* Typing notifications are handled.
* There is an "away" state.
* Logging happens via purple.

Features which are present in the go-whatsapp version but missing here:

* Handling errors consistently (currently logged to console only in some places).
* Geting list of participants in group chat.
* Downloading profile pictures.
* Support for proxy servers.

Known issues:

* If the buddy was not given a local alias or the alias does not match the remote user's friendly name, the notification about them changing their friendly name is shown every time it is received.
* Files received from groups are claimed to originate from the sender rahter than the group. I am undecided whether this is a bug or a feature. The files are downloaded, but link is not shown in group conversation window.

Other planned features:

* Sending proper voice and video messages.
* Join group chat via link.
* Display receipts in conversation window.
* Option to terminate session explicitly.

### Building

#### Pre-Built Binaries

To be announced

#### Instructions

Dependencies: 

* pidgin (libpurple glib gtk)
* pkg-config
* cmake
* make
* go (1.17 or later)
* gcc

This project uses CMake.

    mkdir build
    cd build
    cmake ..
    cmake --build .
    sudo make install/strip

#### Windows Specific

CMake will try to set-up a development environment automatically. 

Additional dependencies:

* [go (32 bit)](https://go.dev/dl/go1.17.5.windows-386.msi)
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
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`. This way, Pidgin's logs look sane.  
  Enter anything but a pipe symbol `|` into the password field. Enable "store password".

* Upon login, a QR code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with WhatsApp Web.
  
* After WhatsApp told us the username, the account will disconnect.  
  The disconnection message contains the password. Pidgin should store it automatically.  
  Reconnect to finish.

#### Environment Variables:

whatsmeow stores all session information in a SQL database. These variables are read when the plug-in loads (even before a connection is established):

* `PURPLE_GOWHATSAPP_DATABASE_DIALECT`  
  default: `sqlite3`  
  File-based sqlite3 database.

* `PURPLE_GOWHATSAPP_DATABASE_ADDRESS`  
  default: `file:purple_user_dir/whatsmeow.db?_foreign_keys=on`  
  Folder must exist, `whatsmeow.db` is created automatically.
  The file-system must support file locking and be responsive, else the database driver will not work. Network shares (NFS, SMB) will probably not work well.
  
Other [SQLDrivers](https://github.com/golang/go/wiki/SQLDrivers) may be added upon request.

#### Purple Settings

* `qrcode-size`  
  The size of the QR code shown for login purposes, in pixels.
  If set to 0, QR code will be rendered as a text message (for text-only clients).
  
* `fetch-contacts`  
  If set to true, buddy list will be populated with contacts sent by server. 
  This is useful for the first login, especially.
  
* `fake-online`  
  If set to true, contacts currently not online will be regarded as "away" (so they still appear in the buddy list).
  If set to false, offline contacts will be regarded as "offline" (no messages can be sent).

* `send-receipt`  
  Selects when to send receipts "double blue tick" notifications:
    * "immediately": immediately upon message receival
    * "on-interact": as the user interacts with the conversation window (currently buggy)
    * "on-answer": as soon as the user sends an answer
    * "never": never

### Notes

#### Attachment Handling and Memory Consumption

Attachments (images, videos, voice messages, stickers, document) are *always* downloaded as *soon as the message is processed*. The user is then asked where they want the file to be written. During this time, the file data is residing in memory multiple times:

* in the input buffer
* in the decryption buffer
* in the go → C message buffer
* in the output buffer

On systems with many concurrent connection, this could exhaust memory.

As of writing, whatsmeow does not offer an interface to read the file in chunks.
