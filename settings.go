package main

import(
    "net/http"
    "io/ioutil"
    "encoding/json"
)

type Settings struct {
    Books []string `json:"books"`
}

var(
    settingsPage []byte
    settings Settings
)

func init() {
    var err error
    settingsPage, err = ioutil.ReadFile("html/settings.htm")
    if err != nil {
        log.Fatal("Failed to read 'settings.htm':", err)
    }
}

func applySettingsHandler(res http.ResponseWriter, req *http.Request) {
    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        res.WriteHeader(500)
        return
    }

    err := json.Unmarshal(body, &settings)
    if err != nil {
        res.WriteHeader(500)
        return
    }

    log.Println(settings)
}

func settingsHandler(res http.ResponseWriter, req *http.Request) {
    res.Write(settingsPage)
}

