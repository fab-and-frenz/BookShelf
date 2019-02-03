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

    flag.StringVar(&certFile, "c", "", "The location of your ssl certificate")
    flag.StringVar(&privKeyFile, "p", "", "The location of your ssl private key")
    flag.Parse()

    if certFile == "" {
        log.Fatal("No certificate specified")
    }
    if privKeyFile == "" {
        log.Fatal("No private key specified")
    }

    httpsMux := http.NewServeMux()

    httpsMux.HandleFunc( "/html/",        htmlHandler         )
    httpsMux.HandleFunc( "/register",     registerPageHandler )
    httpsMux.HandleFunc( "/registeruser", registerUserHandler )
    httpsMux.HandleFunc( "/login",        loginPageHandler    )
    httpsMux.HandleFunc( "/loginuser",    loginUserHandler    )
    httpsMux.HandleFunc( "/library",      libraryHandler      )

    httpsHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        res.Header().Add("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        httpsMux.ServeHTTP(res, req)
    })

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

    e := make(chan error)

    go func() {
        e <- http.ListenAndServe(httpAddr, http.HandlerFunc(tlsRedirectHandler))
    }()

    go func() {
        e <- httpsServer.ListenAndServeTLS(certFile, privKeyFile)
    }()

    log.Println(<-e)
}

func tlsRedirectHandler(res http.ResponseWriter, req *http.Request) {
    host := strings.Split(req.Host, ":")[0]
    port := strings.Split(httpsAddr, ":")[1]
    path := req.URL.Path

    dest := fmt.Sprintf("https://%s:%s%s", host, port, path)
    log.Printf("Redirecting HTTP client '%s' to %s", req.RemoteAddr, dest)
    http.Redirect(res, req, dest, http.StatusTemporaryRedirect)
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

