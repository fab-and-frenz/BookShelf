package main

import(
    "github.com/fabiocolacio/liblit/cbz"
    "net/http"
    "encoding/base64"
    "html/template"
    pth "path"
    "log"
)

type Book struct {
    Type           string `json:"type"`
    Filename       string `json:"filename"`
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
            Title: pth.Base(path),
            Filename: pth.Base(path),
            Cover: base64.StdEncoding.EncodeToString(pages[0]),
            Id: id,
        }

        books = append(books, book)
    }

    t.Execute(res, books)
}

