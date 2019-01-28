package main

import(
    "github.com/fabiocolacio/liblit/cbz"
    "net/http"
    "encoding/base64"
    "html/template"
    pth "path"
    "log"
)

const (
    Cbz = iota
    Epub
    Pdf
)

type Book struct {
    Type           int    `json:"type"`
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
        var book Book
       
        if pth.Ext(path) == ".cbz" {
            book.Type = Cbz
            pages, err := cbz.NewFromFile(path)
            if err != nil {
                res.WriteHeader(500)
                log.Println(err)
                return
            }
            book.Cover = base64.StdEncoding.EncodeToString(pages[0])
        }

        if pth.Ext(path) == ".pdf" {
            book.Type = Pdf
        }

        if pth.Ext(path) == ".epub" {
            book.Type = Epub
        }

        book.Title =  pth.Base(path)
        book.Filename = pth.Base(path)
        book.Id = id

        books = append(books, book)
    }

    t.Execute(res, books)
}

