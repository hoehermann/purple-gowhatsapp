package main

import (
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
	"os"
	// popular alternative: _ "github.com/mattn/go-sqlite3"
)

func init_(purple_user_dir string) int {
	dbLog := waLog.Stdout("Database", "DEBUG", true) // TODO: have logger write to purple
	dialect := os.Getenv("PURPLE_GOWHATSAPP_DATABASE_DIALECT")
	if dialect == "" {
		dialect = "sqlite"
	}
	uri := os.Getenv("PURPLE_GOWHATSAPP_DATABASE_ADDRESS")
	if uri == "" {
		dirname := purple_user_dir + "/whatsmeow" // assume default configuration
		os.Mkdir(dirname, os.ModeDir)             // create directory
		uri = "file:" + dirname + "/store.db?_foreign_keys=on"
	}
	dbLog.Infof("%s connecting to %s", dialect, uri)
	container_, err := sqlstore.New(dialect, uri, dbLog)
	if err != nil {
		dbLog.Errorf("%v", err)
		return 0
	}
	container = container_
	return 1
}
