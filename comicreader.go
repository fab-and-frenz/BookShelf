package main

import(
    "net/http"
    "strconv"
    "encoding/json"
    "html/template"
    "github.com/fabiocolacio/liblit/cbz"
)

func readHandler(res http.ResponseWriter, req *http.Request) {
    id, err := strconv.Atoi(req.URL.Query()["id"][0])
    if err != nil {
        res.WriteHeader(500)
        return
    }

    t, err := template.ParseFiles("html/view.htm")
    if err != nil {
        res.WriteHeader(500)
        return
    }

    idStruct := struct{ Id int }{ Id: id }

    t.Execute(res, idStruct)
}

func getPagesHandler(res http.ResponseWriter, req *http.Request) {
    id, err := strconv.Atoi(req.URL.Query()["id"][0])
    if err != nil {
        res.WriteHeader(500)
        return
    }

    if id >= len(settings.Books) {
        res.WriteHeader(400)
        return
    }

    path := settings.Books[id]
    
    pages, err := cbz.NewFromFile(path)
    if err != nil {
        res.WriteHeader(500)
        return
    }

    payload, err := json.Marshal(pages)
    if err != nil {
        res.WriteHeader(400)
        return
    }
    
    res.Write(payload)
}

