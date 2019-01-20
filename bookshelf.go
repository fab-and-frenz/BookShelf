package main

import(
    "log"
    "os"
    "flag"
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

    http.HandleFunc("/applysettings", applySettingsHandler)
    http.HandleFunc("/settings", settingsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

