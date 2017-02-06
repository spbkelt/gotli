package main

import (
	"github.com/couchbase/gocb"
	"hash/fnv"
	"github.com/pilu/go-base62"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"os"
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

func CreateURLHTTPHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// есть доступ до бакета
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), 405)
			return
		}

		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		var request RequestResponse
		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			http.Error(w, "Test error", 400)
			return
		}

		var longUrl = request.URL

		// Check if url already exists in the database
		var token = hash(longUrl)
		//var shortUrl = "http://localhost:8000/" +  token
		var shortUrl = "http://jetb.co/" +  token

		_, err = bucket.Insert(token, &longUrl,0)

		if err != nil {

			if err == gocb.ErrKeyExists{
				js:=RequestResponse{URL:shortUrl}
				json.NewEncoder(w).Encode(js)
				return
			}

			log.Fatalf("ERROR INSERTING TO BUCKET:%s", err.Error())
		}

		js:=RequestResponse{URL:shortUrl}
		json.NewEncoder(w).Encode(js)

		return
	}
}

func RedirectURLHTTPHandler(bucket *gocb.Bucket) func (w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		token := mux.Vars(r)["token"]

		var content interface{}
		var longUrl string

		bucket.Get(token,&content)

		if content != nil {
			longUrl = content.(string)

		}

		if longUrl != "" {
			http.Redirect(w,r,longUrl,301)
			return
		}

		return
	}
}

func serveHTMLTemplateHandler() func (w http.ResponseWriter, r *http.Request){
	return func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w,r,"static/index.html")
	}
}

func main(){

	couchAddress := os.Getenv("COUCHBASE_ADDRESS")

	if couchAddress  == "" {
		couchAddress  = "127.0.0.1"
	}

	// Connect to Cluster
	cluster, err := gocb.Connect("couchbase://" + couchAddress)

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
	r.HandleFunc("/{token:[a-zA-Z0-9]+}", RedirectURLHTTPHandler(bucket))
	r.HandleFunc("/", serveHTMLTemplateHandler() )

	http.Handle("/", r)

	panic(http.ListenAndServe(":8000",  nil))


}



