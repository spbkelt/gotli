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
	"encoding/json"
)

type RequestResponse  struct {
	URL string
}

//Need for one way convert any string to uniq integer sequence
func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return base62.Encode(int(h.Sum32()))
}

func lookupLongUrlByToken(bucket *gocb.Bucket, token string) (longUrl string, err error) {
	var content interface{}
	_, err = bucket.Get(token,&content)

	if err != nil {
		fmt.Printf(err.Error())
		return "",err
	}
	longUrl = reflect.ValueOf(content).String()
	return longUrl, nil
}

func CreateURLHTTPHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// есть доступ до бакета
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), 405)
			return
		}

		var request RequestResponse

		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		var longUrl = request.URL

		// Check if url already exists in the database
		var token = hash(longUrl)
		var shortUrl = "http://localhost:8000/" +  token

		_, err = bucket.Insert(token, &longUrl,0)

		if err != nil {

			if gocb.ErrKeyExists != nil {
				_,err = bucket.Get(token, longUrl)
				json.NewEncoder(w).Encode(shortUrl)
				return

			}

			log.Fatalf("ERROR INSERTING TO BUCKET:%s", err.Error())
		}

		return
	}
}

func RedirectURLHTTPHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
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

	bucket, err := cluster.OpenBucket("default", "")

	if err != nil {
		log.Fatalf("ERROR OPENING BUCKET:%s", err)
	}

	bucket.Manager("", "").CreatePrimaryIndex("", true, false)

	r := mux.NewRouter()
	r.HandleFunc("/create",CreateURLHTTPHandler(bucket))
	r.HandleFunc("/go/[a-Z0-9]", RedirectURLHTTPHandler(bucket))

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)


}



