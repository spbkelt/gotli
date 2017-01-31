package main

import (
	"github.com/couchbase/gocb"
	"hash/fnv"
	"github.com/pilu/go-base62"
	"log"
	"net/http"
	"reflect"
	"strings"
	"fmt"
	"errors"

)

// bucket reference - reuse as bucket reference in the application
var bucket *gocb.Bucket

type Url struct {
	LongUrl string `json:"long_url"`
}

type AppContext struct {
	bucket        *gocb.Bucket

}

type AppHandler struct {
	*AppContext
	H func(*AppContext, http.ResponseWriter, *http.Request) (int, error)
}

func (ah AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Updated to pass ah.appContext as a parameter to our handler type.
	status, err := ah.H(ah.AppContext, w, r)
	if err != nil {
		log.Printf("HTTP %d: %q", status, err)
		switch status {
		case http.StatusNotFound:
			http.NotFound(w, r)
		// And if we wanted a friendlier error page, we can
		// now leverage our context instance - e.g.
		// err := ah.renderTemplate(w, "http_404.tmpl", nil)
		case http.StatusInternalServerError:
			http.Error(w, http.StatusText(status), status)
		default:
			http.Error(w, http.StatusText(status), status)
		}
	}
}

//Need for convert any string to uniq integer sequence
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// Shortens a given URL passed through in the request.
// If the URL has already been shortened, returns the existing URL.
// Writes the short URL in plain text to w.

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the url parameter has been sent along (and is not empty)
	longUrl := r.URL.Query().Get("url")

	if longUrl == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// Check if url already exists in the database
	var token = base62.Encode(int(hash(longUrl)))
	var value interface{}

	var shortUrl = "http://jetb.co/" +  token

	_, err := bucket.Get(token, &value)

	if err != nil {
		// The URL already doesnt exist! Upsert document with urls

		var urls Url

		urls.LongUrl = longUrl

		_,err = bucket.Upsert(token, urls, 0)

		if err != nil {
			log.Fatalf("ERROR UPSERTING TO BUCKET:%s", err.Error())
		}

		w.Write([]byte(shortUrl))
		return
	}

	return

}

func lookupLongUrlByToken(bucket *gocb.Bucket, token string) (longUrl string, err error) {
	fragment, err := bucket.LookupIn(token).Get("long_url").Execute()

	if err != nil {
		fmt.Printf(err.Error())
		return "",err
	}

	var content interface{}

	err = fragment.Content("long_url",&content )

	if err != nil {
		fmt.Printf(err.Error())
		return "",err
	}

	longUrl = reflect.ValueOf(content).String()
	return longUrl, nil
}

func RedirectHandler(a *AppContext, w http.ResponseWriter, r *http.Request) (int,error){
	// Check if the url parameter has been sent along (and is not empty)
	path := string(r.URL.Path)

	if path == "" {
		http.Error(w, "", http.StatusBadRequest)
		return 400, errors.New("no token found")
	}

	token := strings.TrimPrefix(path,"/")

	var longUrl,err = lookupLongUrlByToken(c.bucket,token)

	if err == nil {
		http.Redirect(w,r,longUrl,301)
		return 301,nil
	}

	return 500,err

}

func main(){

	// Connect to Cluster
	cluster, err := gocb.Connect("couchbase://127.0.0.1")

	if err != nil {
		log.Fatalf("ERROR CONNECTING TO CLUSTER:%s", err)
	}

	bucket,  _ := cluster.OpenBucket("default", "")
	bucket.Manager("", "").CreatePrimaryIndex("", true, false)

	context := &AppContext{bucket: bucket}

	http.HandleFunc("/create",ShortenHandler)
	http.HandleFunc("/", AppHandler{context, RedirectHandler}))
	http.ListenAndServe(":8000", nil)


}



