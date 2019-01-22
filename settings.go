package main

import(
    "log"
    "net/http"
    "path/filepath"
    "encoding/json"
    "io/ioutil"
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
        log.Println("Failed to read request body:", err)
        return
    }

    if err = json.Unmarshal(body, &settings); err != nil {
        res.WriteHeader(500)
        log.Println("Failed to unmarshal settings JSON:", err)
        return
    }

    if err = ioutil.WriteFile(filepath.Join(configDir, settingsFile), body, 0666); err != nil {
        res.WriteHeader(500)
        log.Println("Failed to save settings file:", err)
        return
    }
}

func settingsHandler(res http.ResponseWriter, req *http.Request) {
    res.Write(settingsPage)
}

