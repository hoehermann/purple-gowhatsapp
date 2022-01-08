package main

import (
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"os"
)

/*
 * whatsmew stores all data in a container (set by init_()).
 */
var container *sqlstore.Container = nil

/*
 * Initializes the whatsmew session store.
 * This sets the global container variable.
 */
func init_(purple_user_dir string) int {
	dbLog := PurpleLogger("Handler")
	dialect := os.Getenv("PURPLE_GOWHATSAPP_DATABASE_DIALECT")
	if dialect == "" {
		dialect = "sqlite3"
	}
	uri := os.Getenv("PURPLE_GOWHATSAPP_DATABASE_ADDRESS")
	if uri == "" {
		uri = "file:" + purple_user_dir + "/whatsmeow.db?_foreign_keys=on"
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
