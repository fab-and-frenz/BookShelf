package main

import(
    "log"
    "os"
    "flag"
    "io/ioutil"
    "net/http"
    "github.com/fab-and-frenz/bookshelf/util"
)

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

    http.HandleFunc("/html/", func (res http.ResponseWriter, req *http.Request) {
        data, err := ioutil.ReadFile(req.URL.String())
        if err != nil {
            res.WriteHeader(400)
            return
        }

        res.Write(data)
    })

    http.HandleFunc("/getpages", getPagesHandler)
    http.HandleFunc("/read", readHandler)
    http.HandleFunc("/getbooks", getBooksHandler)
    http.HandleFunc("/library", libraryHandler)
    http.HandleFunc("/applysettings", applySettingsHandler)
    http.HandleFunc("/settings", settingsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

