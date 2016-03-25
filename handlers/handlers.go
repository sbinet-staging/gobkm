package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/tbellembois/gobkm/models"
	"github.com/tbellembois/gobkm/types"

	log "github.com/Sirupsen/logrus"
)

// Env is a structure used to pass objects throughout the application
type Env struct {
	DB            models.Datastore
	GoBkmProxyURL string // the application URL
	TplData       []byte // template data
	CssData       []byte // css data
	JsData        []byte // js data
}

type folderAndBookmark struct {
	Flds          []*types.Folder
	Bkms          []*types.Bookmark
	CssData       string
	JsData        string
	GoBkmProxyURL string
}

type newFolderStruct struct {
	FolderId    int64
	FolderTitle string
}

type newBookmarkStruct struct {
	BookmarkId      int64
	BookmarkURL     string
	BookmarkFavicon []byte
}

func failHTTP(w http.ResponseWriter, functionName string, errorMessage string, httpStatus int) {

	log.Error("%s: %s", functionName, errorMessage)
	w.WriteHeader(httpStatus)
	fmt.Fprint(w, errorMessage)

}

func (env *Env) UpdateBookmarkFavicon(bkm *types.Bookmark) {

	if u, err := url.Parse(bkm.URL); err == nil {

		// bookmark domain name
		bkmDomain := u.Scheme + "://" + u.Host

		// using Google to retrieve the favicon
		faviconRequestUrl := "http://www.google.com/s2/favicons?domain_url=" + bkmDomain

		log.WithFields(log.Fields{
			"bkmDomain":         bkmDomain,
			"faviconRequestUrl": faviconRequestUrl,
		}).Debug("UpdateBookmarkFavicon")

		// performing the request
		if response, err := http.Get(faviconRequestUrl); err == nil {

			defer response.Body.Close()

			log.WithFields(log.Fields{
				"response.ContentLength": response.ContentLength,
			}).Debug("UpdateBookmarkFavicon")

			// converting the image into a base64 string
			image, _ := ioutil.ReadAll(response.Body)
			bkm.Favicon = base64.StdEncoding.EncodeToString(image)

			log.WithFields(log.Fields{
				"bkm": bkm,
			}).Debug("UpdateBookmarkFavicon")

			// updating the bookmark into the DB
			env.DB.UpdateBookmark(bkm)

			// datastore error check
			if err = env.DB.FlushErrors(); err != nil {

				log.WithFields(log.Fields{
					"err": err,
				}).Error("UpdateBookmarkFavicon")

			}
		}

	}
}

func (env *Env) AddBookmarkHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var destinationFolderId int
	var err error
	var js []byte // the returned JSON

	// GET parameters retrieval
	bookmarkUrl := r.URL.Query()["bookmarkUrl"]
	destinationFolderIdParam := r.URL.Query()["destinationFolderId"]

	log.WithFields(log.Fields{
		"bookmarkUrl":              bookmarkUrl,
		"destinationFolderIdParam": destinationFolderIdParam,
	}).Debug("AddBookmarkHandler:Query parameter")

	// parameters check
	if len(bookmarkUrl) == 0 || len(destinationFolderIdParam) == 0 {

		failHTTP(w, "AddBookmarkHandler", "bookmarkUrl empty", http.StatusBadRequest)
		return

	}

	// destinationFolderId convertion
	if destinationFolderId, err = strconv.Atoi(destinationFolderIdParam[0]); err != nil {

		failHTTP(w, "AddBookmarkHandler", "destinationFolderId Atoi conversion", http.StatusInternalServerError)
		return
	}

	// getting the destination folder
	dstFld := env.DB.GetFolder(destinationFolderId)
	// creating a new Bookmark
	newBookmark := types.Bookmark{Title: bookmarkUrl[0], URL: bookmarkUrl[0], Folder: dstFld}
	// saving the bookmark into the DB, getting its id
	bookmarkId := env.DB.SaveBookmark(&newBookmark)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "AddBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	// building the JSON result
	if js, err = json.Marshal(newBookmarkStruct{BookmarkId: bookmarkId, BookmarkURL: bookmarkUrl[0]}); err != nil {

		failHTTP(w, "AddBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	// updating the bookmark favicon
	newBookmark.Id = int(bookmarkId)
	go env.UpdateBookmarkFavicon(&newBookmark)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func (env *Env) AddFolderHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var err error
	var js []byte // the returned JSON

	// GET parameters retrieval
	folderName := r.URL.Query()["folderName"]

	log.WithFields(log.Fields{
		"folderName": folderName,
	}).Debug("AddFolderHandler:Query parameter")

	// parameters check
	if len(folderName) == 0 {

		failHTTP(w, "AddFolderHandler", "bookmarkUrl empty", http.StatusBadRequest)
		return

	}

	// creating a new Folder
	newFolder := types.Folder{Title: folderName[0]}
	// saving the folder into the DB, getting its id
	folderId := env.DB.SaveFolder(&newFolder)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "AddFolderHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	if js, err = json.Marshal(newFolderStruct{FolderId: folderId, FolderTitle: folderName[0]}); err != nil {

		failHTTP(w, "AddFolderHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func (env *Env) DeleteFolderHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var err error
	var folderId int

	// GET parameters retrieval
	folderIdParam := r.URL.Query()["folderId"]

	log.WithFields(log.Fields{
		"folderIdParam": folderIdParam,
	}).Debug("DeleteFolderHandler:Query parameter")

	// parameters check
	if len(folderIdParam) == 0 {

		failHTTP(w, "DeleteFolderHandler", "folderIdParam empty", http.StatusBadRequest)
		return

	}

	// folderId convertion
	if folderId, err = strconv.Atoi(folderIdParam[0]); err != nil {

		failHTTP(w, "DeleteFolderHandler", "folderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the folder
	fld := env.DB.GetFolder(folderId)
	// deleting it
	env.DB.DeleteFolder(fld)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "DeleteFolderHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) DeleteBookmarkHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var err error
	var bookmarkId int

	// GET parameters retrieval
	bookmarkIdParam := r.URL.Query()["bookmarkId"]

	log.WithFields(log.Fields{
		"bookmarkIdParam": bookmarkIdParam,
	}).Debug("DeleteBookmarkHandler:Query parameter")

	// parameters check
	if len(bookmarkIdParam) == 0 {

		failHTTP(w, "DeleteBookmarkHandler", "bookmarkIdParam empty", http.StatusBadRequest)
		return

	}

	// bookmarkId convertion
	if bookmarkId, err = strconv.Atoi(bookmarkIdParam[0]); err != nil {

		failHTTP(w, "DeleteBookmarkHandler", "bookmarkId Atoi conversion", http.StatusInternalServerError)
		return
	}

	// getting the bookmark
	bkm := env.DB.GetBookmark(bookmarkId)
	// deleting it
	env.DB.DeleteBookmark(bkm)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "DeleteBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) RenameFolderHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var folderId int
	var err error

	// GET parameters retrieval
	folderIdParam := r.URL.Query()["folderId"]
	folderName := r.URL.Query()["folderName"]

	log.WithFields(log.Fields{
		"folderId":   folderId,
		"folderName": folderName,
	}).Debug("RenameFolderHandler:Query parameter")

	// parameters check
	if len(folderIdParam) == 0 || len(folderName) == 0 {

		failHTTP(w, "RenameFolderHandler", "folderId or folderName empty", http.StatusBadRequest)
		return

	}

	// folderId convertion
	if folderId, err = strconv.Atoi(folderIdParam[0]); err != nil {

		failHTTP(w, "RenameFolderHandler", "folderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the folder
	fld := env.DB.GetFolder(folderId)
	// renaming it
	fld.Title = folderName[0]
	// updating the folder into the DB
	env.DB.UpdateFolder(fld)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "AddBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) RenameBookmarkHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var bookmarkId int
	var err error

	// GET parameters retrieval
	bookmarkIdParam := r.URL.Query()["bookmarkId"]
	bookmarkName := r.URL.Query()["bookmarkName"]

	log.WithFields(log.Fields{
		"bookmarkId":   bookmarkId,
		"bookmarkName": bookmarkName,
	}).Debug("RenameBookmarkHandler:Query parameter")

	// parameters check
	if len(bookmarkIdParam) == 0 || len(bookmarkName) == 0 {

		failHTTP(w, "RenameBookmarkHandler", "bookmarkId or bookmarkName empty", http.StatusBadRequest)
		return

	}

	// bookmarkId convertion
	if bookmarkId, err = strconv.Atoi(bookmarkIdParam[0]); err != nil {

		failHTTP(w, "RenameBookmarkHandler", "bookmarkId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the bookmark
	bkm := env.DB.GetBookmark(bookmarkId)
	// renaming it
	bkm.Title = bookmarkName[0]
	// updating the folder into the DB
	env.DB.UpdateBookmark(bkm)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "RenameBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) MoveBookmarkHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var bookmarkId int
	var destinationFolderId int
	var err error

	// GET parameters retrieval
	bookmarkIdParam := r.URL.Query()["bookmarkId"]
	destinationFolderIdParam := r.URL.Query()["destinationFolderId"]

	log.WithFields(log.Fields{
		"bookmarkIdParam":          bookmarkIdParam,
		"destinationFolderIdParam": destinationFolderIdParam,
	}).Debug("MoveBookmarkHandler:Query parameter")

	// parameters check
	if len(bookmarkIdParam) == 0 || len(destinationFolderIdParam) == 0 {

		failHTTP(w, "MoveBookmarkHandler", "bookmarkIdParam or destinationFolderIdParam empty", http.StatusBadRequest)
		return

	}

	// bookmarkId and destinationFolderId convertion
	if bookmarkId, err = strconv.Atoi(bookmarkIdParam[0]); err != nil {

		failHTTP(w, "MoveBookmarkHandler", "bookmarkId Atoi conversion", http.StatusInternalServerError)
		return

	}
	if destinationFolderId, err = strconv.Atoi(destinationFolderIdParam[0]); err != nil {

		failHTTP(w, "MoveBookmarkHandler", "destinationFolderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the bookmark
	bkm := env.DB.GetBookmark(bookmarkId)

	// and the destination folder if it exists
	if destinationFolderId != 0 {

		dstFld := env.DB.GetFolder(destinationFolderId)

		log.WithFields(log.Fields{
			"srcBkm": bkm,
			"dstFld": dstFld,
		}).Debug("MoveBookmarkHandler: retrieved Folder instances")

		// updating the source folder parent
		bkm.Folder = dstFld

	} else {

		bkm.Folder = nil

	}

	// updating the folder into the DB
	env.DB.UpdateBookmark(bkm)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "MoveBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) MoveFolderHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var sourceFolderId int
	var destinationFolderId int
	var err error

	// GET parameters retrieval
	sourceFolderIdParam := r.URL.Query()["sourceFolderId"]
	destinationFolderIdParam := r.URL.Query()["destinationFolderId"]

	log.WithFields(log.Fields{
		"sourceFolderIdParam":      sourceFolderIdParam,
		"destinationFolderIdParam": destinationFolderIdParam,
	}).Debug("GetChildrenFoldersHandler:Query parameter")

	// parameters check
	if len(sourceFolderIdParam) == 0 || len(destinationFolderIdParam) == 0 {

		failHTTP(w, "MoveFolderHandler", "sourceFolderIdParam or destinationFolderIdParam empty", http.StatusBadRequest)
		return

	}

	// sourceFolderId and destinationFolderId convertion
	if sourceFolderId, err = strconv.Atoi(sourceFolderIdParam[0]); err != nil {

		failHTTP(w, "MoveFolderHandler", "sourceFolderId Atoi conversion", http.StatusInternalServerError)
		return

	}
	if destinationFolderId, err = strconv.Atoi(destinationFolderIdParam[0]); err != nil {

		failHTTP(w, "MoveFolderHandler", "destinationFolderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the source folder instance
	srcFld := env.DB.GetFolder(sourceFolderId)

	// and the destination folder if it exists
	if destinationFolderId != 0 {

		dstFld := env.DB.GetFolder(destinationFolderId)

		log.WithFields(log.Fields{
			"srcFld": srcFld,
			"dstFld": dstFld,
		}).Debug("MoveFolderHandler: retrieved Folder instances")

		//updating the source folder parent
		srcFld.Parent = dstFld

	} else {

		srcFld.Parent = nil

	}

	// updating the source folder into the DB
	env.DB.UpdateFolder(srcFld)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "MoveBookmarkHandler", err.Error(), http.StatusInternalServerError)
		return

	}

}

func (env *Env) GetFolderBookmarksHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var folderId int
	var err error
	var js []byte // the returned JSON

	// GET parameters retrieval
	folderIdParam := r.URL.Query()["folderId"]

	log.WithFields(log.Fields{
		"folderIdParam": folderIdParam,
	}).Debug("GetFolderBookmarksHandler:Query parameter")

	// parameters check
	if len(folderIdParam) == 0 {

		failHTTP(w, "GetFolderBookmarksHandler", "folderIdParam empty", http.StatusBadRequest)
		return

	}

	// folderId convertion
	if folderId, err = strconv.Atoi(folderIdParam[0]); err != nil {

		failHTTP(w, "GetFolderBookmarksHandler", "folderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the folder bookmarks
	bkms := env.DB.GetFolderBookmarks(folderId)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "GetFolderBookmarksHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	// adding them into a map
	bookmarksMap := make([]*types.Bookmark, 0)

	for _, bkm := range bkms {

		bookmarksMap = append(bookmarksMap, bkm)
	}

	if js, err = json.Marshal(bookmarksMap); err != nil {

		failHTTP(w, "GetFolderBookmarksHandler", err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func (env *Env) GetChildrenFoldersHandler(w http.ResponseWriter, r *http.Request) {

	// vars
	var folderId int
	var err error
	var js []byte // the returned JSON

	// GET parameters retrieval
	folderIdParam := r.URL.Query()["folderId"]

	log.WithFields(log.Fields{
		"folderIdParam": folderIdParam,
	}).Debug("GetChildrenFoldersHandler:Query parameter")

	if len(folderIdParam) == 0 {

		failHTTP(w, "GetChildrenFoldersHandler", "folderIdParam empty", http.StatusBadRequest)
		return

	}

	// folderId convertion
	if folderId, err = strconv.Atoi(folderIdParam[0]); err != nil {

		failHTTP(w, "GetChildrenFoldersHandler", "folderId Atoi conversion", http.StatusInternalServerError)
		return

	}

	// getting the folder children folders
	flds := env.DB.GetChildrenFolders(folderId)

	// datastore error check
	if err = env.DB.FlushErrors(); err != nil {

		failHTTP(w, "GetChildrenFoldersHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	// adding them into a map
	foldersMap := make([]*types.Folder, 0)

	for _, fld := range flds {

		foldersMap = append(foldersMap, fld)

	}

	if js, err = json.Marshal(foldersMap); err != nil {

		failHTTP(w, "GetChildrenFoldersHandler", err.Error(), http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (env *Env) MainHandler(w http.ResponseWriter, r *http.Request) {

	var folderAndBookmark = new(folderAndBookmark)

	bkms := env.DB.GetRootBookmarks()
	flds := env.DB.GetRootFolders()

	if err := env.DB.FlushErrors(); err != nil {
		log.Panic(err)
	}

	folderAndBookmark.Bkms = bkms
	folderAndBookmark.Flds = flds
	folderAndBookmark.CssData = string(env.CssData)
	folderAndBookmark.JsData = string(env.JsData)
	folderAndBookmark.GoBkmProxyURL = env.GoBkmProxyURL

	htmlTpl := template.New("main")
	htmlTpl.Parse(string(env.TplData))

	htmlTpl.Execute(w, folderAndBookmark)

	for _, bkm := range bkms {
		log.WithFields(log.Fields{
			"Title": bkm.Title,
			"URL":   bkm.URL,
		}).Info("Handler:found bookmark")
	}
	for _, fld := range flds {
		log.WithFields(log.Fields{
			"Title": fld.Title,
		}).Info("Handler:found folder")
	}

}