package main

import(
    "net/http"
    "html/template"
    "log"
)

func loginPageHandler(res http.ResponseWriter, req *http.Request) {
    t, err := template.ParseFiles("html/auth.html")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if err = t.Execute(res, "Login"); err != nil {
        log.Println(err)
        return
    }
}
