package main

import(
    "net/http"
    "log"
    "bytes"
    "html/template"
    "encoding/json"
    "io/ioutil"
    "context"
    "github.com/mongodb/mongo-go-driver/bson"
    "github.com/mongodb/mongo-go-driver/bson/primitive"
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

const(
    Epub = iota
    Mobi
    Pdf
    Cbz
    Cbr
    Cbt
    Txt
    Md
)

type Book struct {
    Id         primitive.ObjectID `json:"id"       bson:"_id"`

    Type       int                `json:"type"     bson:"type"`

    Title      string             `json:"title"    bson:"title"`
    Authors  []string             `json:"authors"  bson:"authors"`
    Genre      string             `json:"genre"    bson:"genre"`

    Filename   string             `json:"filename" bson:"filename"`
    Data     []byte               `json:"data"     bson:"data"`
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

    cursor, err := booksBucket.Find(bson.D{}, options.GridFSFind())
    ctx := context.Background()
    defer cursor.Close(ctx)

    if err != nil {
        log.Println(err)
    }
    
    var books []Book
    for cursor.Next(ctx) {
        var book Book
        if err := cursor.Decode(&book); err != nil {
            log.Println(err)
        }
        books = append(books, book)
    }

    if err := t.Execute(res, books); err != nil {
        log.Println(err)
    }
}

func uploadBookHandler(res http.ResponseWriter, req *http.Request) {
    data, err := ioutil.ReadAll(req.Body)
    if err != nil {
        res.WriteHeader(500)
        return
    }
    
    var bookFile Book
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

