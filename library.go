package main

import(
    "github.com/fabiocolacio/liblit/cbz"
    "net/http"
    "encoding/json"
)

type Book struct {
    Type           string `json:"type"`
    Title          string `json:"title"`
    Author         string `json:"author"`
    Contributors   string `json:"contributors"`
    Subjects       string `json:"subjects"`
    Cover        []byte   `json:"cover"`
}

func getBooksHandler(res http.ResponseWriter, req *http.Request) {
    var books []Book
    for _, path := range settings.Books {
        pages, err := cbz.NewFromFile(path)
        if err != nil {
            res.WriteHeader(500)
            return
        }

        book := Book {
            Type: "comic",
            Cover: pages[0],
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
