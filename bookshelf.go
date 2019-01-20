package main

import(
    "log"
    "os"
    "flag"
    "net/http"
    "io/ioutil"
    "github.com/fab-and-frenz/bookshelf/util"
)

var(
    settingsPage []byte
)

func init() {
    var err error
    settingsPage, err = ioutil.ReadFile("html/settings.htm")
    if err != nil {
        log.Fatal("Failed to read 'settings.htm':", err)
    }
}

func main() {
    var configDir string

    flag.StringVar(&configDir, "c", "", "The directory containing configuration files for BookShelf")
    flag.Parse()

    if configDir == "" {
        log.Println("No configuration directory has been specified!")
        return
    }

    if !util.FileExists(configDir) {
        if err := os.Mkdir(configDir, 0666); err != nil {
            log.Printf("Failed to create directory '%s': %s", configDir, err.Error())
            return
        }
    }

    http.HandleFunc("/settings", SettingsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func SettingsHandler(res http.ResponseWriter, req *http.Request) {
    res.Write(settingsPage)
}

