package main

import(
    "github.com/go-chi/chi"
    "github.com/go-chi/chi/middleware"
    "github.com/go-chi/jwtauth"
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
    sr := chi.NewRouter()

    sr.Use(middleware.Logger)
    sr.Use(middleware.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains"))

    // Connect Unprotected Routes to their Handlers
    sr.Get ( "/html/",        htmlHandler         )
    sr.Get ( "/register",     registerPageHandler )
    sr.Post( "/registeruser", registerUserHandler )
    sr.Get ( "/login",        loginPageHandler    )
    sr.Post( "/loginuser",    loginUserHandler    )
    sr.Get ( "/",             loginPageHandler    )

    // Protected Routes that Require JWT authentication
    // You must be logged in to access these routes
    sr.Group(func(pr chi.Router) {
        pr.Use(jwtauth.Verifier(tokenAuth))
        pr.Use(jwtauth.Authenticator)

        pr.Get ( "/logout",       logoutHandler       )
        pr.Get ( "/library",      libraryHandler      )
        pr.Post( "/uploadbook",   uploadBookHandler   )
        pr.Get ( "/downloadbook", downloadBookHandler )
        pr.Get ( "/read",         readBookHandler     )
    })

    // Setup the HTTPS Server
    httpsServer := &http.Server {
        Addr: httpsAddr,
        Handler: sr,
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

