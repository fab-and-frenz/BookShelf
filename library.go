package main

import(
    "net/http"
    "log"
    "html/template"
    "encoding/hex"
    "io"
    "context"
    "github.com/go-chi/jwtauth"
    "github.com/mongodb/mongo-go-driver/bson"
    "github.com/mongodb/mongo-go-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/gridfs"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var(
    booksBucket *gridfs.Bucket
)

// Max size a user can upload at once
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

// libraryHandler shows the html page with a user's library
func libraryHandler(res http.ResponseWriter, req *http.Request) {
    _, claims, err := jwtauth.FromContext(req.Context())
    if err != nil {
        log.Println(err)
        res.WriteHeader(500)
        return
    }

    t, err := template.ParseFiles("html/library.htm")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    user, err := lookupUser(claims["username"].(string))
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if err := t.Execute(res, user.Books); err != nil {
        log.Println(err)
    }
}

// readBookHandler shows the page that displays a book for reading
// (when you press "read" on a book from library).
func readBookHandler(res http.ResponseWriter, req *http.Request) {
    _, claims, err := jwtauth.FromContext(req.Context())
    if err != nil {
        log.Println(err)
        res.WriteHeader(500)
        return
    }

    // Get the book's id from the request
    idQuery := req.URL.Query().Get("id")
    var id primitive.ObjectID
    if _, err := hex.Decode(id[:], []byte(idQuery)); err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }

    // Check if the user owns the book with the requested id
    if !userOwnsBook(claims["username"].(string), id) {
        res.WriteHeader(400)
        log.Println("No such book")
        return
    }
   
    // Load the epub-viewer html (all that is supported currently)
    t, err := template.ParseFiles("html/epub-viewer.html")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    // Serve the webpage
    t.Execute(res, Book{Id: id})
}

// uploadBookHandler responsds to requests to upload a book to the server
func uploadBookHandler(res http.ResponseWriter, req *http.Request) {
    _, claims, err := jwtauth.FromContext(req.Context())
    if err != nil {
        log.Println(err)
        res.WriteHeader(500)
        return
    }

    // Parse the uploaded forms
    if err := req.ParseMultipartForm(maxFormMemory); err != nil {
        res.WriteHeader(500)
        return
    }

    // Upload each uploaded book to the server
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
            bson.M{"username": claims["username"].(string)},
            bson.M{"$push": bson.M{"books": Book{Id: stream.FileID.(primitive.ObjectID), Title: filename}}},
            options.Update().SetUpsert(true),
        )
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }
    }

    // Go back to the library page 
    http.Redirect(res, req, "/library", 302)
}

// downloadBookHandler sends a user a book to download
func downloadBookHandler(res http.ResponseWriter, req *http.Request) {
    _, claims, err := jwtauth.FromContext(req.Context())
    if err != nil {
        log.Println(err)
        res.WriteHeader(500)
        return
    }
  
    // Get the book's id from the request
    idQuery := req.URL.Query().Get("id")
    var id primitive.ObjectID
    if _, err := hex.Decode(id[:], []byte(idQuery)); err != nil {
        res.WriteHeader(400)
        log.Println(err)
        return
    }

    // Check if the user owns the book
    if !userOwnsBook(claims["username"].(string), id) {
        res.WriteHeader(400)
        log.Println("No such book")
        return
    }

    // Load the book data from the database
    stream, err := booksBucket.OpenDownloadStream(id)
    defer stream.Close()
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    // Send the book data to the user
    if _, err := io.Copy(res, stream); err != nil {
        log.Println(err)
        return
    }
}

// userOwnsBook checks if a user with username owns the book specified by id.
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

