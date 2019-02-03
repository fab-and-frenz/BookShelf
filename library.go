package main

import(
    "net/http"
    "log"
    "bytes"
    "html/template"
    "encoding/json"
    "io/ioutil"
    "github.com/mongodb/mongo-go-driver/mongo/gridfs"
    "github.com/mongodb/mongo-go-driver/mongo/options"
)

var(
    booksBucket *gridfs.Bucket
)

func init() {
    var err error
    booksBucket, err = gridfs.NewBucket(db, options.GridFSBucket().SetName("books"))
    if err != nil {
        log.Fatal(err)
    }
}

type BookFile struct {
    Filename   string `json:"filename" bson:"filename"`
    Data     []byte   `json:"data" bson:"data"`
}

func libraryHandler(res http.ResponseWriter, req *http.Request) {
    _, err := checkSession(req)
    if err != nil {
        res.Write([]byte("oopsie woopsie, you need to log in"))
        log.Println(err)
        return
    }

    t, err := template.ParseFiles("html/library.htm")
    if err != nil {
        res.WriteHeader(500)
        return
    }

    t.Execute(res, nil)
}

func uploadBookHandler(res http.ResponseWriter, req *http.Request) {
    data, err := ioutil.ReadAll(req.Body)
    if err != nil {
        res.WriteHeader(500)
        return
    }
    
    var bookFile BookFile
    if err = json.Unmarshal(data, &bookFile); err != nil {
        res.WriteHeader(400)
        return
    }

    if _, err := booksBucket.UploadFromStream(bookFile.Filename, bytes.NewReader(bookFile.Data), options.GridFSUpload()); err != nil {
        res.WriteHeader(400)
        return
    }

    res.WriteHeader(200)
}

