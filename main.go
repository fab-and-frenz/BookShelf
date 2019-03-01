package main

import(
    "net/http"
    "log"
    "io/ioutil"
    "crypto/tls"
    "strings"
    "flag"
    "fmt"
)

const(
    httpAddr  string = ":8080"
    httpsAddr string = ":9090"
)

func main() {
    var (
        certFile    string
        privKeyFile string
    )

    // Parse command-line arguments
    flag.StringVar(&certFile, "c", "", "The location of your ssl certificate")
    flag.StringVar(&privKeyFile, "p", "", "The location of your ssl private key")
    flag.Parse()

    // Exit if no certificate or private key were specified.
    if certFile == "" {
        log.Fatal("No certificate specified")
    }
    if privKeyFile == "" {
        log.Fatal("No private key specified")
    }

    // Connect each route to a corresponding request-handler function
    httpsMux := http.NewServeMux()
    httpsMux.HandleFunc( "/html/",        htmlHandler         )
    httpsMux.HandleFunc( "/register",     registerPageHandler )
    httpsMux.HandleFunc( "/registeruser", registerUserHandler )
    httpsMux.HandleFunc( "/login",        loginPageHandler    )
    httpsMux.HandleFunc( "/loginuser",    loginUserHandler    )
    httpsMux.HandleFunc( "/logout",       logoutHandler       )
    httpsMux.HandleFunc( "/library",      libraryHandler      )
    httpsMux.HandleFunc( "/uploadbook",   uploadBookHandler   )
    httpsMux.HandleFunc( "/downloadbook", downloadBookHandler )
    httpsMux.HandleFunc( "/read",         readBookHandler     )
    httpsMux.HandleFunc( "/",             loginPageHandler    )

    // The request handler for all https requests
    httpsHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        // Adds the HSTS header to all https requests
        res.Header().Add("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

        // Give the correct response for a given request
        httpsMux.ServeHTTP(res, req)
    })

    // Setup the HTTPS Server
    httpsServer := &http.Server {
        Addr: httpsAddr,
        Handler: httpsHandler,
        TLSConfig: &tls.Config {
            MinVersion: tls.VersionTLS12,
            PreferServerCipherSuites: true,
            CipherSuites: []uint16 {
                tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
                tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
                tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            },
        },
    }

    // Create a channel to listen for erros
    e := make(chan error)

    // Handle HTTP requests in a separate thread
    go func() {
        e <- http.ListenAndServe(httpAddr, http.HandlerFunc(tlsRedirectHandler))
    }()

    // Handle HTTPS requests in a separate thread
    go func() {
        e <- httpsServer.ListenAndServeTLS(certFile, privKeyFile)
    }()

    // Wait for one of the threads to exit before quitting
    log.Println(<-e)
}

// Redirect HTTP requests to HTTPS
func tlsRedirectHandler(res http.ResponseWriter, req *http.Request) {
    host := strings.Split(req.Host, ":")[0]
    port := strings.Split(httpsAddr, ":")[1]
    path := req.URL.Path

    dest := fmt.Sprintf("https://%s:%s%s", host, port, path)
    log.Printf("Redirecting HTTP client '%s' to %s", req.RemoteAddr, dest)
    http.Redirect(res, req, dest, http.StatusTemporaryRedirect)
}

// Honor requests for files inside of the /html folder
func htmlHandler(res http.ResponseWriter, req *http.Request) {
    data, err := ioutil.ReadFile(req.URL.String()[1:])
    if err != nil {
        log.Println(err)
        res.WriteHeader(400)
        return
    }
    res.Write(data)
}

