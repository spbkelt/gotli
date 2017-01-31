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
	"github.com/gorilla/mux"
)

// bucket reference - reuse as bucket reference in the application
var bucket *gocb.Bucket

type Url struct {
	LongUrl string `json:"long_url"`
}


//Need for one way convert any string to uniq integer sequence
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
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

func CreateUrlHttpHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// есть доступ до бакета
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), 405)
			return
		}

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
}

func RedirectUrlHttpHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// есть доступ до бакета
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), 405)
			return
		}

		// Check if the url parameter has been sent along (and is not empty)
		path := string(r.URL.Path)

		if path == "" {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		token := strings.TrimPrefix(path,"/")

		var longUrl,err = lookupLongUrlByToken(bucket,token)

		if err == nil {
			http.Redirect(w,r,longUrl,301)
			return
		}

		return
	}
}

func main(){

	// Connect to Cluster
	cluster, err := gocb.Connect("couchbase://127.0.0.1")

	if err != nil {
		log.Fatalf("ERROR CONNECTING TO CLUSTER:%s", err)
	}

	bucket,  _ := cluster.OpenBucket("default", "")
	bucket.Manager("", "").CreatePrimaryIndex("", true, false)

	r := mux.NewRouter()
	r.HandleFunc("/create",CreateUrlHttpHandler(bucket))
	r.HandleFunc("/go/[a-Z0-9]", RedirectUrlHttpHandler(bucket))

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)


}



