package main

import(
    "net/http"
    "log"
)

func libraryHandler(res http.ResponseWriter, req *http.Request) {
    _, err := checkSession(req)
    if err != nil {
        res.Write([]byte("oopsie woopsie, you need to log in"))
        log.Println(err)
        return
    }

    res.Write([]byte("you are here"))
}

