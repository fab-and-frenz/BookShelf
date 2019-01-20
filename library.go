package main

import(
    "github.com/fabiocolacio/liblit/cbz"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "log"
)

type Book struct {
    Type           string `json:"type"`
    Title          string `json:"title"`
    Author         string `json:"author"`
    Contributors   string `json:"contributors"`
    Subjects       string `json:"subjects"`
    Cover        []byte   `json:"cover"`
    Id             int    `json:"id"`
}

var(
    libraryPage []byte
)

func init() {
    var err error
    libraryPage, err = ioutil.ReadFile("html/library.htm")
    if err != nil {
        log.Fatal("Failed to load library.htm")
    }
}

func libraryHandler(res http.ResponseWriter, req *http.Request) {
    res.Write(libraryPage)        
}

func getBooksHandler(res http.ResponseWriter, req *http.Request) {
    var books []Book
    for id, path := range settings.Books {
        pages, err := cbz.NewFromFile(path)
        if err != nil {
            res.WriteHeader(500)
            return
        }

        book := Book {
            Type: "comic",
            Cover: pages[0],
            Id: id,
        }

        books = append(books, book)
    }

    payload, err := json.Marshal(books)
    if err != nil {
        res.WriteHeader(500)
        return
    }

    res.Write(payload)
}

