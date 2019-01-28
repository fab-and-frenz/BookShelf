package main

import(
    "net/http"
    "log"
    "strconv"
    "io/ioutil"
)

func downloadHandler(res http.ResponseWriter, req *http.Request) {
    id, err := strconv.Atoi(req.URL.Query()["id"][0])
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if id > len(settings.Books) {
        res.WriteHeader(500)
        log.Printf("Book with index %d does not exist.", id)
        return
    }

    path := settings.Books[id]

    data, err := ioutil.ReadFile(path)
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    res.Write(data)
}

