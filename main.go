package main

import (
	"github.com/couchbase/gocb"
	"hash/fnv"
	"github.com/pilu/go-base62"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
)

type RequestResponse  struct {
	URL string
}

const listenerPort = "8000"

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
			http.Error(w, "Malformed user's request", 400)
			return
		}

		var longUrl = request.URL

		// Check if url already exists in the database
		var token = hash(longUrl)
		//var shortUrl = "http://localhost:8000/" +  token
		var shortUrl = "https://jetb.co/" +  token

		_, err = bucket.Insert(token, &longUrl,0)

		if err != nil && gocb.ErrKeyExists != nil{
			http.Error(w, "Service is temporarily unavailable:" + err.Error(), 503)
			return
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

		_, err := bucket.Get(token,&content)

		if err == gocb.ErrKeyNotFound {
			http.Error(w, "Missing token key in database: " + token, 400)
			return
		}

		if err != nil && err != gocb.ErrKeyNotFound{
			http.Error(w, "Internal server error:" + err.Error(), 503)
			return
		}


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

	log.Printf("Starting program...")
	couchAddress := "couchbase.default.svc.cluster.local"
	//couchAddress := "a56e28553f30511e6a7ea029336298d7-1093911673.eu-central-1.elb.amazonaws.com"
	// Connect to Cluster
	cluster, err := gocb.Connect("couchbase://" + couchAddress)

	if err != nil {
		log.Fatalf("ERROR CONNECTING TO CLUSTER:%s", err.Error())

	}

	log.Printf("Connected to cluster service endpoint:" + couchAddress)

	bucket, err := cluster.OpenBucket("default","")

	if err != nil {
		log.Fatalf("ERROR OPENING BUCKET:%s", err.Error())
	}

	log.Printf("Opened bucket default")

	err = bucket.Manager("", "").CreatePrimaryIndex("", true, false)

	if err != nil {
		log.Fatalf("ERROR CREATING PRIMARY INDEX:%s", err.Error())
	}

	log.Printf("Created primary index in bucket default")

	r := mux.NewRouter()

	r.HandleFunc("/create",CreateURLHTTPHandler(bucket))
	r.HandleFunc("/{token:[a-zA-Z0-9]+}", RedirectURLHTTPHandler(bucket))
	r.HandleFunc("/", serveHTMLTemplateHandler() )

	http.Handle("/", r)

	panic(http.ListenAndServe(":"+listenerPort,  nil))

	log.Printf("Started listener on:" + listenerPort)


}



