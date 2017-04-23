package bookmarks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
	"strings"
)


// =======================================================================================
// #mark Object Definitions
// =======================================================================================


type BookmarkPresenter struct {
	Name string   `json:"name"`
	Url  string   `json:"url"`
	Tags []string `json:"tags"`
}

func (b BookmarkPresenter) IsBookmarkValid() bool {
	return (len(b.Name) > 0) && (len(b.Url) > 0) && (len(b.Tags) > 0)
}

func (b BookmarkPresenter) toBookmarkModel() BookmarkModel {
	return BookmarkModel{Name: b.Name, Url: b.Url, Tags: b.Tags}
}

type ErrorPresenter struct {
	ErrorMessageSummary string `json:"summary"`
	ErrorMessageDetails string `json:"error_details"`
}

type ApiKeyModel struct {
	KeyValue string
	Company  string
}

type BookmarkModel struct {
	Name string
	Url  string
	Tags []string
}


// =======================================================================================
// #mark Endpoint Methods
// =======================================================================================


func handleEndpointBookmarks(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	log.Infof(ctx, r.Method)

	if r.Method == "POST" {
		handleEndpointBookmarksPOST(w, r)
	} else {
		handleEndpointBookmarksGET(w, r)
	}

}


func handleEndpointBookmarksGET(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	q := datastore.NewQuery("Bookmark")
    t := q.Run(ctx)
    
	var currentBookmark BookmarkModel
	dataIterator := newDatastoreIterator(t)
	
	var outputArray []BookmarkPresenter
	for dataIterator.NextBookmark( &currentBookmark ) {
		log.Infof( ctx, "looping" )
		outputArray = append( outputArray, presentBookmark( currentBookmark ) )
	}
	
	if dataIterator.currentError != nil {
		handleError(w, dataIterator.currentError, http.StatusInternalServerError)
		return		
	}

	bookmarkBuffer, _ := toJSONBytesBuffer(outputArray)
	fmt.Fprint(w, bookmarkBuffer.String())
}


// POSTs into this system are only allowed by users with an API key
// so check that first, then save the record
func handleEndpointBookmarksPOST(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	var isValid bool
	var err error

	if isValid, err = isApiKeyValid(ctx, r.Header.Get("apikey")); err != nil {
		var errCode int
		if isValid == false {
			errCode = http.StatusForbidden
		} else {
			errCode = http.StatusInternalServerError
		}

		handleError(w, err, errCode)
		return
	}

	if isValid {
		requestedBody, _ := ioutil.ReadAll(r.Body)
		jsonDecoder := json.NewDecoder(strings.NewReader(string(requestedBody)))

		var inBookmark BookmarkPresenter
		if err := jsonDecoder.Decode(&inBookmark); err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		if inBookmark.IsBookmarkValid() {
			var bookmark BookmarkModel

			bookmark = inBookmark.toBookmarkModel()
			k := datastore.NewIncompleteKey(ctx, "Bookmark", nil)
			if _, err := datastore.Put(ctx, k, &bookmark); err != nil {
				handleError(w, err, http.StatusInternalServerError)
				return
			}

			bookmarkBuffer, _ := toJSONBytesBuffer(inBookmark)
			fmt.Fprint(w, bookmarkBuffer.String())

		} else {
			handleError(w, errors.New("Bookmark failed validation check, must have all of: name, url, tags"), http.StatusBadRequest)
			return
		}
	}

}


func bootstrap(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	initialKey := ApiKeyModel{KeyValue: "CHANGE ME", Company: "Wilcox Development Solutions"}
	if _, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "ApiKey", nil), &initialKey); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}


// =======================================================================================
// #mark Helper Methods
// =======================================================================================


func handleError(w http.ResponseWriter, err error, httpErrCode int) {
	w.WriteHeader(httpErrCode)

	errorBuffer, _ := toJSONBytesBuffer(presentError(err))
	fmt.Fprintf(w, errorBuffer.String())
}


// isApiKeyValid checks if a given API key exists in the database / is authorized for write ops
// It returns boolean, error.
// If no record is found for the given API key, an error is returned
func isApiKeyValid(ctx context.Context, key string) (bool, error) {
	q := datastore.NewQuery("ApiKey").
		Filter("KeyValue =", key).
		Limit(1)

	t := q.Run(ctx)
	var x ApiKeyModel

	log.Infof(ctx, key)

	_, err := t.Next(&x)
	if err == datastore.Done {
		return false, errors.New("API Key not found")
	}

	return true, err
}


func toJSONBytesBuffer(inThing interface{}) (bytes.Buffer, error) {
	var buffer bytes.Buffer

	thingDataBytes, err := json.Marshal(inThing)
	if err != nil {
		return buffer, err
	}
	buffer.Write(thingDataBytes)
	buffer.WriteString("\n")

	return buffer, err
}


func presentError(inErr error) ErrorPresenter {
	output := ErrorPresenter{ErrorMessageSummary: inErr.Error()}

	return output
}


func presentBookmark(inBookmark BookmarkModel) BookmarkPresenter {
	return BookmarkPresenter{Name: inBookmark.Name, Url: inBookmark.Url, Tags: inBookmark.Tags}
}

type datastoreIterator struct {
	allGone bool
	cursor *datastore.Iterator
	currentError error
}

func newDatastoreIterator( in *datastore.Iterator ) *datastoreIterator {
	current := new(datastoreIterator)
	current.cursor = in
	current.allGone = false
	
	return current
}


func (it *datastoreIterator) handleError( err error ) {
	if ( err == datastore.Done ) {
		it.allGone = true
		err = nil
	}
	
	if ( err != nil ) {
		it.currentError = err
		it.allGone = true
	}
}


// datastore package does interesting / fancy type work here to make sure we're putting
// data into a good record value. Thus we can NOT just use interface{} here
func (it *datastoreIterator) NextBookmark( bookmark *BookmarkModel ) (bool) {
	//thing, _ := (it.currentRecord).Value( BookmarkModel )
	_, err := it.cursor.Next(bookmark)
	it.handleError( err )
		
	return it.allGone == false
}


// =======================================================================================
// #mark Init Functions
// =======================================================================================


func setupRoutes() {
	http.HandleFunc("/bookmarks", handleEndpointBookmarks)
	http.HandleFunc("/bootstrap", bootstrap)
}


func init() {
	setupRoutes()
}
