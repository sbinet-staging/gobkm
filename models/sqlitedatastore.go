package models

import (
	"database/sql"

	"github.com/tbellembois/gobkm/types"

	log "github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
)

const (
	dbdriver = "sqlite3"
)

// SQLiteDataStore implements the Datastore interface
// to store the Bookmarks in SQLite3
type SQLiteDataStore struct {
	*sql.DB
	err error
}

// NewDBstore returns a database connection to the given dataSourceName
// ie. a path to the sqlite database file
func NewDBstore(dataSourceName string) (*SQLiteDataStore, error) {

	log.WithFields(log.Fields{
		"dataSourceName": dataSourceName,
	}).Debug("NewDBstore:params")

	var db *sql.DB
	var err error

	if db, err = sql.Open(dbdriver, dataSourceName); err != nil {

		log.WithFields(log.Fields{
			"dataSourceName": dataSourceName,
		}).Error("NewDBstore:error opening the database")

		return nil, err

	}

	return &SQLiteDataStore{db, nil}, nil

}

func (db *SQLiteDataStore) FlushErrors() error {

	lastError := db.err

	db.err = nil

	return lastError

}

// CreateDatabase creates the database tables
func (db *SQLiteDataStore) CreateDatabase() {

	if db.err != nil {
		return
	}

	if _, db.err = db.Exec("PRAGMA foreign_keys = ON"); db.err != nil {

		log.Error("CreateDatabase: error executing the PRAGMA request")
		return

	}

	if _, db.err = db.Exec("CREATE TABLE IF NOT EXISTS folder ( id integer PRIMARY KEY, title string NOT NULL, parentFolderId integer, nbChildrenFolders integer, FOREIGN KEY (parentFolderId) references folder(id) ON DELETE CASCADE)"); db.err != nil {

		log.Error("CreateDatabase: error executing the CREATE TABLE request for table bookmark")
		return

	}

	if _, db.err = db.Exec("CREATE TABLE IF NOT EXISTS bookmark ( id integer PRIMARY KEY, title string NOT NULL, url string NOT NULL, favicon string, folderId integer, FOREIGN KEY (folderId) references folder(id) ON DELETE CASCADE)"); db.err != nil {

		log.Error("CreateDatabase: error executing the CREATE TABLE request for table bookmark")
		return

	}

}

// PopulateDatabase populate the database with sample folders and bookmarks
func (db *SQLiteDataStore) PopulateDatabase() {

	if db.err != nil {
		return
	}

	var count int

	db.err = db.QueryRow("SELECT COUNT(*) as count FROM folder").Scan(&count)

	if count > 0 {
		return
	}

	var folders []*types.Folder
	var bookmarks []*types.Bookmark

	folder1 := types.Folder{Id: 1, Title: "IT"}
	folder2 := types.Folder{Id: 2, Title: "Development", Parent: &folder1}

	bookmark1 := types.Bookmark{Id: 1, Title: "GoLang", URL: "https://golang.org/", Favicon: "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAABHNCSVQICAgIfAhkiAAAAb9JREFUOI3tkj9oU1EUh797c3lgjA4xL61FX0yhMQqmW5QgFim4+GcyQ3Hp1MlBqFIyOGUobRScnYoQikNA0Ao6WJS2UIdiK7SUVGtfIZg0iMSA+Iy5Dg9fGnyLu2e6nHPu9zv3/K7QWuMXjfqebjQbOM5PIuEjHI6Ywq9P/TlUdm09+3KeNxtlAHbLWzTrNeTBQxjhHuLHohrgwqkBRi5dpO+4JQDEh80NfePOXaIDJ3FigximBUAyk+5SOvFphR/tNovvyzg769TKmxQLecS5a9d1dOQ2zp7N6bjF1PAZlJKMv1hFpVxIa+0t96+cBWD82TLr2zaGaVGbvYcEqLx+gmFajKZiqANBeo/2MZcb89RHUzEAeiNh5nJjGKZF9VUJAFks5FGVrc7IuuW7VH518slMGlHdpljII/sTSW+7j5ohEIrP9S9cnnxIaShOaSjOzNoOBNz81ceLHqg/kRRqv0ggGGLCdm3t+fqRmZtZ15HKEhN2Go1ABUO06VjfBdDSLQS0IFNd4fytSQAWHuR4B8gW7lWJP8B7rtA8zU7zfH4V8f0brew0ou37j/wBHigx2D2d/LvHJ/Vv8R8AvwHjjZMncK4ImgAAAABJRU5ErkJggg==", Folder: &folder2}

	folders = append(folders, &folder1)
	folders = append(folders, &folder2)

	bookmarks = append(bookmarks, &bookmark1)

	for _, fld := range folders {
		db.SaveFolder(fld)
	}

	for _, bkm := range bookmarks {
		db.SaveBookmark(bkm)
	}

	return

}

// GetBookmark returns a Bookmark instance with the given id
func (db *SQLiteDataStore) GetBookmark(id int) *types.Bookmark {

	log.WithFields(log.Fields{
		"id": id,
	}).Debug("GetBookmark")

	if db.err != nil {
		return nil
	}

	var folderId sql.NullInt64
	bkm := new(types.Bookmark)

	db.err = db.QueryRow("SELECT id, title, url, favicon, folderId FROM bookmark WHERE id=?", id).Scan(&bkm.Id, &bkm.Title, &bkm.URL, &bkm.Favicon, &folderId)

	switch {

	case db.err == sql.ErrNoRows:
		log.WithFields(log.Fields{
			"id": id,
		}).Debug("GetBookmark:no bookmark with that ID")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetBookmark:SELECT query error")
		return nil

	default:
		log.WithFields(log.Fields{
			"Id":       bkm.Id,
			"Title":    bkm.Title,
			"folderId": folderId,
		}).Debug("GetBookmark:bookmark found")

		// retrieving the parent
		if folderId.Int64 != 0 {

			bkm.Folder = db.GetFolder(int(folderId.Int64))

			if db.err != nil {
				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetBookmark:parent Folder retrieving error")
				return nil
			}

		}

	}

	return bkm
}

// GetFolder returns a Folder instance with the given id
func (db *SQLiteDataStore) GetFolder(id int) *types.Folder {

	log.WithFields(log.Fields{
		"id": id,
	}).Debug("GetFolder")

	if db.err != nil || id == 0 {
		return nil
	}

	var parentFldId sql.NullInt64
	fld := new(types.Folder)

	db.err = db.QueryRow("SELECT id, title, parentFolderId FROM folder WHERE id=?", id).Scan(&fld.Id, &fld.Title, &parentFldId)

	switch {

	case db.err == sql.ErrNoRows:
		log.WithFields(log.Fields{
			"id": id,
		}).Debug("GetFolder:no folder with that ID")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetFolder:SELECT query error")
		return nil

	default:
		log.WithFields(log.Fields{
			"Id":          fld.Id,
			"Title":       fld.Title,
			"parentFldId": parentFldId,
		}).Debug("GetFolder:folder found")

		// recursively retrieving the parents
		if parentFldId.Int64 != 0 {

			fld.Parent = db.GetFolder(int(parentFldId.Int64))

		}
	}

	return fld
}

// GetRootBookmarks the root bookmarks (with no folder)
func (db *SQLiteDataStore) GetRootBookmarks() []*types.Bookmark {

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT id, title, url, favicon FROM bookmark WHERE folderId is null ORDER BY title")

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetRootBookmarks:no root bookmarks")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetRootBookmarks:SELECT query error")
		return nil

	default:

		bkms := make([]*types.Bookmark, 0)

		for rows.Next() {

			bkm := new(types.Bookmark)

			db.err = rows.Scan(&bkm.Id, &bkm.Title, &bkm.URL, &bkm.Favicon)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetRootBookmarks:error scanning the query result row")
				return nil

			}

			bkms = append(bkms, bkm)

		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetRootBookmarks:error looping rows")
			return nil

		}

		return bkms
	}
}

// GetNoIconBookmarks returns the bookmarks with no favicon
func (db *SQLiteDataStore) GetNoIconBookmarks() []*types.Bookmark {

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT id, title, url, favicon FROM bookmark WHERE favicon='' ORDER BY title")

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetNoIconBookmarks:no bookmarks")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetNoIconBookmarks:SELECT query error")
		return nil

	default:

		bkms := make([]*types.Bookmark, 0)

		for rows.Next() {

			bkm := new(types.Bookmark)

			db.err = rows.Scan(&bkm.Id, &bkm.URL, &bkm.Title, &bkm.Favicon)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetNoIconBookmarks:error scanning the query result row")
				return nil

			}

			bkms = append(bkms, bkm)

		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetNoIconBookmarks:error looping rows")
			return nil

		}

		return bkms

	}

}

// GetAllBookmarks returns all the bookmarks as an array of *Bookmark
func (db *SQLiteDataStore) GetAllBookmarks() []*types.Bookmark {

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT * FROM bookmark ORDER BY title")

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetAllBookmarks:no bookmarks")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetAllBookmarks:SELECT query error")
		return nil

	default:

		bkms := make([]*types.Bookmark, 0)

		for rows.Next() {

			bkm := new(types.Bookmark)
			var fldId sql.NullInt64

			db.err = rows.Scan(&bkm.Id, &bkm.Title, &bkm.URL, &fldId)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetAllBookmarks:error scanning the query result row")
				return nil

			}

			bkm.Folder = db.GetFolder(int(fldId.Int64))

			bkms = append(bkms, bkm)
		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetAllBookmarks:error looping rows")
			return nil

		}

		return bkms
	}
}

// GetFolderBookmarks returns the bookmarks of the given folder id
func (db *SQLiteDataStore) GetFolderBookmarks(id int) []*types.Bookmark {

	log.WithFields(log.Fields{
		"id": id,
	}).Debug("GetFolderBookmarks")

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT id, title, url, favicon, folderId FROM bookmark WHERE folderId is ? ORDER BY title", id)

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetFolderBookmarks:no bookmarks")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetFolderBookmarks:SELECT query error")
		return nil

	default:

		bkms := make([]*types.Bookmark, 0)

		for rows.Next() {

			bkm := new(types.Bookmark)
			var parentFldId sql.NullInt64

			db.err = rows.Scan(&bkm.Id, &bkm.Title, &bkm.URL, &bkm.Favicon, &parentFldId)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetFolderBookmarks:error scanning the query result row")
				return nil

			}

			bkms = append(bkms, bkm)

		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetFolderBookmarks:error looping rows")
			return nil

		}

		return bkms

	}

}

// GetChildrenFolders returns the children folders as an array of *Folder
func (db *SQLiteDataStore) GetChildrenFolders(id int) []*types.Folder {

	log.WithFields(log.Fields{
		"id": id,
	}).Debug("GetChildrenFolders")

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT id, title, parentFolderId, nbChildrenFolders FROM folder WHERE parentFolderId is ? ORDER BY title", id)

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetChildrenFolders:no folders")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetChildrenFolders:SELECT query error")
		return nil

	default:

		flds := make([]*types.Folder, 0)

		for rows.Next() {

			fld := new(types.Folder)
			var parentFldId sql.NullInt64

			db.err = rows.Scan(&fld.Id, &fld.Title, &parentFldId, &fld.NbChildrenFolders)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetChildrenFolders:error scanning the query result row")
				return nil

			}

			flds = append(flds, fld)

		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetChildrenFolders:error looping rows")
			return nil

		}

		return flds

	}

}

// GetRootFolders returns the root folders as an array of *Folder
func (db *SQLiteDataStore) GetRootFolders() []*types.Folder {

	if db.err != nil {
		return nil
	}

	var rows *sql.Rows

	rows, db.err = db.Query("SELECT id, title, parentFolderId, nbChildrenFolders FROM folder WHERE parentFolderId is null ORDER BY title")

	defer rows.Close()

	switch {

	case db.err == sql.ErrNoRows:
		log.Debug("GetRootFolders:no folders")
		return nil

	case db.err != nil:
		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("GetRootFolders:SELECT query error")
		return nil

	default:

		flds := make([]*types.Folder, 0)

		for rows.Next() {

			fld := new(types.Folder)

			var parentFldId sql.NullInt64

			db.err = rows.Scan(&fld.Id, &fld.Title, &parentFldId, &fld.NbChildrenFolders)

			if db.err != nil {

				log.WithFields(log.Fields{
					"err": db.err,
				}).Error("GetRootFolders:error scanning the query result row")
				return nil

			}

			flds = append(flds, fld)

		}

		if db.err = rows.Err(); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("GetRootFolders:error looping rows")
			return nil

		}

		return flds
	}

}

//// hasChildrenFolders returns true if the folder with the given id has children
//func (db *SQLiteDataStore) hasChildrenFolders(id int) (bool, error) {
//
//	log.WithFields(log.Fields{
//		"id": id,
//	}).Debug("hasChildrenFolders:params")
//
//	stmt, err := db.Prepare("SELECT count(*) FROM folder WHERE parentId=?")
//
//	if err != nil {
//		log.WithFields(log.Fields{
//			"err": err,
//		}).Error("hasChildrenFolders:SELECT request prepare error")
//		return false, err
//	}
//
//	defer stmt.Close()
//
//	var count int
//
//	// querying the DB
//	err = stmt.QueryRow(id).Scan(&count)
//
//	return count > 0, nil
//}

// SaveFolder saves the given new Folder into the db and returns the folder id
// called only on folder creation or rename
// so only the Title has to be set
func (db *SQLiteDataStore) SaveFolder(f *types.Folder) int64 {

	log.WithFields(log.Fields{
		"f": f,
	}).Debug("SaveFolder")

	if db.err != nil {
		return 0
	}

	var stmt *sql.Stmt

	// id will be auto incremented
	stmt, db.err = db.Prepare("INSERT INTO folder(title, parentFolderId, nbChildrenFolders) values(?,?,?)")
	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("SaveFolder:SELECT request prepare error")
		return 0

	}

	defer stmt.Close()

	var res sql.Result

	if f.Parent != nil {
		res, db.err = stmt.Exec(f.Title, f.Parent.Id, f.NbChildrenFolders)
	} else {
		res, db.err = stmt.Exec(f.Title, nil, f.NbChildrenFolders)
	}
	id, _ := res.LastInsertId() // we should check the error here too...

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("SaveFolder:INSERT query error")
		return 0

	}

	return id
}

// UpdateBookmark updates the given bookmark
func (db *SQLiteDataStore) UpdateBookmark(b *types.Bookmark) {

	log.WithFields(log.Fields{
		"b": b,
	}).Debug("UpdateBookmark")

	var stmt *sql.Stmt
	var tx *sql.Tx

	tx, db.err = db.Begin()

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("Update bookmark:transaction begin failed")

		return
	}

	stmt, db.err = tx.Prepare("UPDATE bookmark SET title=?, url=?, folderId=?, favicon=? WHERE id=?")

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("Update bookmark:UPDATE request prepare error")

		return
	}

	defer stmt.Close()

	if b.Folder != nil {
		_, db.err = stmt.Exec(b.Title, b.URL, b.Folder.Id, b.Favicon, b.Id)
	} else {
		_, db.err = stmt.Exec(b.Title, b.URL, nil, b.Favicon, b.Id)
	}

	if db.err != nil {

		tx.Rollback()

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateBookmark: UPDATE query error")
		return

	}

	tx.Commit()

}

// SaveBookmark saves the given Bookmark into the db
func (db *SQLiteDataStore) SaveBookmark(b *types.Bookmark) int64 {

	log.WithFields(log.Fields{
		"b": b,
	}).Debug("SaveBookmark")

	var stmt *sql.Stmt

	stmt, db.err = db.Prepare("INSERT INTO bookmark(title, url, folderId, favicon) values(?,?,?,?)")

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("SaveBookmark:INSERT request prepare error")
		return 0

	}

	defer stmt.Close()

	var res sql.Result

	if b.Folder != nil {
		res, db.err = stmt.Exec(b.Title, b.URL, b.Folder.Id, b.Favicon)
	} else {
		res, db.err = stmt.Exec(b.Title, b.URL, nil, b.Favicon)
	}

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("SaveBookmark:INSERT query error")
		return 0

	}

	id, _ := res.LastInsertId()

	return id
}

func (db *SQLiteDataStore) DeleteBookmark(b *types.Bookmark) {

	log.WithFields(log.Fields{
		"b": b,
	}).Debug("DeleteBookmark")

	_, db.err = db.Exec("DELETE from bookmark WHERE id=?", b.Id)

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("DeleteBookmark:DELETE query error")
		return

	}

	return
}

func (db *SQLiteDataStore) UpdateFolder(f *types.Folder) {

	log.WithFields(log.Fields{
		"f": f,
	}).Debug("UpdateFolder")

	var oldParentFolderId sql.NullInt64

	// retrieving the parentFolderId of the folder to be updated
	if db.err = db.QueryRow("SELECT parentFolderId from folder WHERE id=?", f.Id).Scan(&oldParentFolderId); db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateFolder:SELECT query error")
		return

	}

	log.WithFields(log.Fields{
		"oldParentFolderId": oldParentFolderId,
		"f.Parent":          f.Parent,
	}).Debug("UpdateFolder")

	var stmt *sql.Stmt

	stmt, db.err = db.Prepare("UPDATE folder SET title=?, parentFolderId=?, nbChildrenFolders=(SELECT count(*) from folder WHERE parentFolderId=?) WHERE id=?")

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateFolder:UPDATE request prepare error")

		return
	}

	defer stmt.Close()

	if f.Parent != nil {
		_, db.err = stmt.Exec(f.Title, f.Parent.Id, f.Id, f.Id)
	} else {
		_, db.err = stmt.Exec(f.Title, nil, f.Id, f.Id)
	}

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateFolder:UPDATE query error")
		return

	}

	// updating the old parent nbChildrenFolders
	stmt, db.err = db.Prepare("UPDATE folder SET nbChildrenFolders=(SELECT count(*) from folder WHERE parentFolderId=?) WHERE id=?")

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateFolder:UPDATE old parent request prepare error")
		return

	}

	defer stmt.Close()

	if _, db.err = stmt.Exec(oldParentFolderId, oldParentFolderId); db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("UpdateFolder:UPDATE old parent request error")
		return
	}

	// and the new
	if f.Parent != nil {
		if _, db.err = stmt.Exec(f.Parent.Id, f.Parent.Id); db.err != nil {

			log.WithFields(log.Fields{
				"err": db.err,
			}).Error("UpdateFolder:UPDATE new parent request error")
			return
		}

	}

}

func (db *SQLiteDataStore) DeleteFolder(f *types.Folder) {

	log.WithFields(log.Fields{
		"f": f,
	}).Debug("DeleteFolder")

	_, db.err = db.Exec("DELETE from folder WHERE id=?", f.Id)

	if db.err != nil {

		log.WithFields(log.Fields{
			"err": db.err,
		}).Error("DeleteFolder:DELETE query error")
		return

	}

	return

}