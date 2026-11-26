# Facebook Messenger (optional, experimental)

The plug-in can also be built with a second protocol, **Facebook (E2EE)** (`prpl-gometa`), for Facebook Messenger, end-to-end encrypted chats included. It is off by default and changes nothing in the WhatsApp protocol when it is off.

Plain chats go through [messagix](https://github.com/mautrix/meta/tree/main/pkg/messagix) from mautrix-meta, encrypted chats through whatsmeow's Facebook client. Both protocols live in this one shared object because a process can host only one Go c-archive, so a separate Go-based Facebook plug-in could not be loaded next to this one.

### Building

    cmake -DGOMETA=ON ..

This adds `-tags gometa` to the Go build (every `gometa*.go` file and `gometabridge.c` carry `//go:build gometa`) and compiles `glue/gometa_init.c` and `gometabridge.c` into the plug-in. Without the option none of it is built.

The Go module needs `go.mau.fi/mautrix-meta` and, as mautrix-meta's own `go.mod` does, a `replace` of `github.com/imroc/req/v3` with beeper's fork. mautrix-meta is pinned to a commit on its main branch: Facebook's two-factor code pages need [mautrix/meta#633](https://github.com/mautrix/meta/pull/633), which is in no release yet. Version selection also raises `github.com/lib/pq` and `github.com/mattn/go-colorable` by a minor version, and `go.mau.fi/util` to a pre-release of v0.11.2, for the WhatsApp build.

### Set-Up

* Create an account with the protocol "Facebook (E2EE)". The username is the e-mail address of the Facebook account, the password its Facebook password.
* At the first login Facebook usually asks for a code (two-factor authentication, or a code sent by e-mail). The plug-in asks for it in a request window. A UI without request windows (no `request_input`) gets a message from a contact called "Facebook" instead; reply to it with the code.
* After a successful login the session cookies are stored in the account (setting `gometa_session`, in `accounts.xml`) and later logins use them instead of the password. If the session has expired, the login fails once and the next one asks for the password and the code again.
* The keys of the encrypted chats are kept in `gometa-e2ee-<username>.db` in the purple user directory. The file is deleted together with the account.

### What works

* Plain and encrypted 1:1 chats, plain group chats (with their members and names), in both directions.
* Messages sent from another device of the same account are shown as sent.
* Pictures are shown inline, other files as a link to the downloaded file. Attachments are downloaded automatically (up to 100 MB) to `gometa/attachments/<username>/` in the purple user directory.
* Sending files, in 1:1 and group chats.
* Replies are shown with the quoted text above them, edits as `[EDIT] …`, deletions and reactions as a line in the chat, the same way the WhatsApp protocol shows them.
* Locations and shared contacts are shown as text.
* Typing notifications in both directions; read receipts are sent when the UI shows a chat as read.

### Limits

* Encrypted group chats are shown as 1:1 chats named after the group's thread.
* Edits, deletions and reactions name the message they are about by id only. They are shown with the message's text only for messages shown since the plug-in was started; the index of those is kept in memory.
* Sending reactions, replies, mentions and group administration (create, add, remove, rename, leave) are not implemented.
* There is no `facebook` icon in `assets/pixmaps`; the protocol names it, so a UI that has one shows it.
* The protocol id `prpl-gometa` is provisional.
* Facebook does not offer an API for this. Logging in from an unknown client can make Facebook ask for further verification or restrict the account.

### Tests

The Go tests need the tag and the C glue of a `-DGOMETA=ON` build (here in `build/`), linked in whole:

    ar rcs build/libgometa-glue.a build/glue/CMakeFiles/whatsmeow.dir/*.o
    go test -tags gometa -run Gometa -ldflags="-extldflags '-Lbuild -Wl,--whole-archive -lgometa-glue -Wl,--no-whole-archive $(pkg-config --libs purple gmodule-2.0)'" .
