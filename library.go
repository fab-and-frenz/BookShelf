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
}

func libraryHandler(res http.ResponseWriter, req *http.Request) {
    jwt, err := checkSession(req)
    if err != nil {
        res.Write([]byte("oopsie woopsie, you need to log in"))
        log.Println(err)
        return
    }

    t, err := template.ParseFiles("html/library.htm")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    user, err := lookupUser(jwt.Username)
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if err := t.Execute(res, user.Books); err != nil {
        log.Println(err)
    }
}

func readBookHandler(res http.ResponseWriter, req *http.Request) {
    idQuery := req.URL.Query().Get("id")
    var id primitive.ObjectID
    if _, err := hex.Decode(id[:], []byte(idQuery)); err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }
    
    t, err := template.ParseFiles("html/epub-viewer.html")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    var book Book
   
    cursor, err := booksBucket.Find(bson.D{{"_id", id}}, options.GridFSFind())
    ctx := context.Background()
    if err != nil {
        res.WriteHeader(404)
        log.Println(err)
        return
    }
    defer cursor.Close(ctx)

    if !cursor.Next(ctx) {
        res.WriteHeader(404)
        log.Println("No book found")
        return
    }

    if err := cursor.Decode(&book); err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    t.Execute(res, book)
}

func uploadBookHandler(res http.ResponseWriter, req *http.Request) {
    jwt, err := checkSession(req)
    if err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }

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
        if err != nil {
            res.WriteHeader(400)
            log.Println(err)
            return
        }
        defer stream.Close()

        if _, err := io.Copy(stream, file); err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }

        _, err = usersCollection.UpdateOne(
            context.Background(),
            bson.M{"username": jwt.Username},
            bson.M{"$push": bson.M{"books": Book{Id: stream.FileID, Title: filename}}},
            options.Update().SetUpsert(true),
        )
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }
    }

    res.WriteHeader(200)
}

func downloadBookHandler(res http.ResponseWriter, req *http.Request) {
    jwt, err := checkSession(req)
    if err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }
    
    idQuery := req.URL.Query().Get("id")
    var id primitive.ObjectID
    if _, err := hex.Decode(id[:], []byte(idQuery)); err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }

    if !userOwnsBook(jwt.Username, id) {
        res.WriteHeader(400)
        log.Println("No such book")
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

func userOwnsBook(username string, id primitive.ObjectID) bool {
    return usersCollection.FindOne(
        context.Background(),
        bson.M{
            "username": username,
            "books": bson.M{
                "$elemMatch": bson.M{
                    "_id": id,
                },
            },
        },
        options.FindOne(),
    ) != nil
}

