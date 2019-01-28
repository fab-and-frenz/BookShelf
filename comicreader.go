package main

import(
    "net/http"
    "strconv"
    "log"
    "encoding/base64"
    "io/ioutil"
    "html/template"
    "github.com/fabiocolacio/liblit/cbz"
)

func readHandler(res http.ResponseWriter, req *http.Request) {
    tp, err := strconv.Atoi(req.URL.Query()["type"][0])
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    id, err := strconv.Atoi(req.URL.Query()["id"][0])
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if tp == Cbz {
        t, err := template.ParseFiles("html/view.htm")
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }

        if id >= len(settings.Books) {
            res.WriteHeader(400)
            log.Println(err)
            return
        }

        path := settings.Books[id]

        pages, err := cbz.NewFromFile(path)
        if err != nil {
            res.WriteHeader(500)
            return
        }

        b64Pages := make([]string, len(pages))
        for i, page := range pages {
            b64Pages[i] = base64.StdEncoding.EncodeToString(page)
        }

        if err = t.Execute(res, b64Pages); err != nil {
            log.Println(err)
        }
    } else if tp == Pdf {
        data, err := ioutil.ReadFile(settings.Books[id])
        if err != nil {
            res.WriteHeader(500)
            log.Println(err)
            return
        }

        res.Write(data)
    }
}

