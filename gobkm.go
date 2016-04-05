package main

//go:generate go-bindata -pkg $GOPACKAGE -o bindata.go static/...

import (
	"flag"
	"net/http"

	"github.com/tbellembois/gobkm/handlers"
	"github.com/tbellembois/gobkm/models"

	log "github.com/Sirupsen/logrus"
)

// DB URL
const (
	dbURL = "./bkm.db"
)

func main() {

	// getting the params
	listenPort := flag.String("port", "8080", "the port to listen")
	goBkmProxyURL := flag.String("proxy", "http://localhost:"+*listenPort, "the proxy full URL if used")
	debug := flag.Bool("debug", false, "debug (verbose log)")

	flag.Parse()

	log.WithFields(log.Fields{
		"listenPort": *listenPort,
		"goBkmURL":   *goBkmProxyURL,
	}).Debug("main:flags")

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	// database init
	datastore, err := models.NewDBstore(dbURL)

	if err != nil {
		log.Panic(err)
	}

	// create DB
	datastore.CreateDatabase()
	datastore.PopulateDatabase()

	// create the environment
	env := handlers.Env{DB: datastore, GoBkmProxyURL: *goBkmProxyURL}

	// initializing the static data
	env.TplMainData, err = Asset("static/main.html")
	if err != nil {
		log.Panic(err)
	}

	env.CssData, err = Asset("static/main.css")
	if err != nil {
		log.Panic(err)
	}

	env.JsData, err = Asset("static/main.js")
	if err != nil {
		log.Panic(err)
	}

	// getting the bookmarks with no favicon
	noIconBookmarks := env.DB.GetNoIconBookmarks()

	// datastore error check
	if err := env.DB.FlushErrors(); err != nil {
		panic(err)
	}

	// updating them
	for _, bkm := range noIconBookmarks {

		//go env.UpdateBookmarkFavicon(bkm)
		env.UpdateBookmarkFavicon(bkm)

	}

	http.HandleFunc("/getChildrenFolders/", env.GetChildrenFoldersHandler)
	http.HandleFunc("/getFolderBookmarks/", env.GetFolderBookmarksHandler)
	http.HandleFunc("/moveFolder/", env.MoveFolderHandler)
	http.HandleFunc("/moveBookmark/", env.MoveBookmarkHandler)
	http.HandleFunc("/renameFolder/", env.RenameFolderHandler)
	http.HandleFunc("/renameBookmark/", env.RenameBookmarkHandler)
	http.HandleFunc("/addFolder/", env.AddFolderHandler)
	http.HandleFunc("/addBookmark/", env.AddBookmarkHandler)
	http.HandleFunc("/deleteFolder/", env.DeleteFolderHandler)
	http.HandleFunc("/deleteBookmark/", env.DeleteBookmarkHandler)
	http.HandleFunc("/export/", env.ExportHandler)
	http.HandleFunc("/import/", env.ImportHandler)
	http.HandleFunc("/", env.MainHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":"+*listenPort, nil)

}
