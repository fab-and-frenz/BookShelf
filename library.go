package main

import(
    "net/http"
    "log"
    "html/template"
    "encoding/hex"
    "io"
    "context"
    "github.com/mongodb/mongo-go-driver/bson"
    "github.com/mongodb/mongo-go-driver/bson/primitive"
    "github.com/mongodb/mongo-go-driver/mongo/gridfs"
    "github.com/mongodb/mongo-go-driver/mongo/options"
)

var(
    booksBucket *gridfs.Bucket
)

const(
    maxFormMemory int64 = 100000000
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
    if err := req.ParseMultipartForm(maxFormMemory); err != nil {
        res.WriteHeader(500)
        return
    }

    for _, header := range req.MultipartForm.File["books"] {
        filename := header.Filename
        file, err := header.Open()
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }

        stream, err := booksBucket.OpenUploadStream(filename, options.GridFSUpload())
        defer stream.Close()
        if err != nil {
            res.WriteHeader(400)
            log.Println(err)
            return
        }

        if _, err := io.Copy(stream, file); err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }
    }

    res.WriteHeader(200)
}

func downloadBookHandler(res http.ResponseWriter, req *http.Request) {
    idQuery := req.URL.Query().Get("id")
    var id primitive.ObjectID
    if _, err := hex.Decode(id[:], []byte(idQuery)); err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }

    stream, err := booksBucket.OpenDownloadStream(id)
    defer stream.Close()
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if _, err := io.Copy(res, stream); err != nil {
        log.Println(err)
        return
    }
}

