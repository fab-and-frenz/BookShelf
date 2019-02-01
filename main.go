package main

import(
    "net/http"
    "log"
    "io/ioutil"
)

func main() {
    http.HandleFunc("/html/", htmlHandler)
    http.HandleFunc("/register", registerPageHandler)
    http.HandleFunc("/registeruser", registerUserHandler)
    http.HandleFunc("/login", loginPageHandler)

    log.Fatal(http.ListenAndServe(":8080", nil))
}

func htmlHandler(res http.ResponseWriter, req *http.Request) {
    data, err := ioutil.ReadFile(req.URL.String()[1:])
    if err != nil {
        log.Println(err)
        res.WriteHeader(400)
        return
    }
    res.Write(data)
}

