package main

import(
    "log"
    "os"
    "flag"
    "io/ioutil"
    "net/http"
    "encoding/json"
    "path/filepath"
    "github.com/fab-and-frenz/bookshelf/util"
)

var(
    configDir string
    settingsFile string = "settings.json"
)

func init() {
    flag.StringVar(&configDir, "c", "", "The directory containing configuration files for BookShelf")
    flag.Parse()

    if configDir == "" {
        log.Fatal("No configuration directory has been specified!")
    }

    if !util.FileExists(configDir) {
        if err := os.Mkdir(configDir, 0666); err != nil {
            log.Fatal("Failed to create config directory.", configDir, err.Error())
        }
    }

    if util.FileExists(filepath.Join(configDir, settingsFile)) {
        data, err := ioutil.ReadFile(filepath.Join(configDir, settingsFile))
        if err != nil {
            log.Println("Failed to read settings file:", err)
        }
        if err = json.Unmarshal(data, &settings); err != nil {
            log.Println("Failed to parse settings data:", err)
        }
    }
}

func main() {
    http.HandleFunc("/html/", func (res http.ResponseWriter, req *http.Request) {
        data, err := ioutil.ReadFile(req.URL.String()[1:])
        if err != nil {
           log.Println(err)
           res.WriteHeader(400)
            return
        }
        res.Write(data)
    })

    http.HandleFunc("/download", downloadHandler)
    http.HandleFunc("/read", readHandler)
    http.HandleFunc("/library", libraryHandler)
    http.HandleFunc("/applysettings", applySettingsHandler)
    http.HandleFunc("/settings", settingsHandler)

    http.HandleFunc("/", libraryHandler)

    log.Fatal(http.ListenAndServe(":8080", nil))
}

