package main

import(
    "github.com/fabiocolacio/liblit/cbz"
    "net/http"
    "encoding/json"
    "encoding/base64"
    "html/template"
    "log"
)

type Book struct {
    Type           string `json:"type"`
    Title          string `json:"title"`
    Author         string `json:"author"`
    Contributors   string `json:"contributors"`
    Subjects       string `json:"subjects"`
    Cover          string `json:"cover"`
    Id             int    `json:"id"`
}

func libraryHandler(res http.ResponseWriter, req *http.Request) {
    t, err := template.ParseFiles("html/library.htm")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    var books []Book
    for id, path := range settings.Books {
        pages, err := cbz.NewFromFile(path)
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }

        book := Book {
            Type: "comic",
            Cover: base64.StdEncoding.EncodeToString(pages[0]),
            Id: id,
        }

        books = append(books, book)
    }

    t.Execute(res, books)
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
            Cover: base64.StdEncoding.EncodeToString(pages[0]),
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

