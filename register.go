package main

import(
    "golang.org/x/crypto/scrypt"
    "crypto/rand"
    "context"
    "net/http"
    "html/template"
    "log"
)

type User struct {
    Username     string
    Salt       []byte
    SaltedHash []byte
}

const(
    scryptN int = 32768
    scryptR int = 8
    scryptP int = 1
    hashLen int = 32
    saltLen int = 16
)

func HashAndSaltPassword(password, salt []byte) ([]byte, error) {
    return scrypt.Key(password, salt, scryptN, scryptR, scryptP, hashLen)
}

func registerPageHandler(res http.ResponseWriter, req *http.Request) {
    t, err := template.ParseFiles("html/auth.html")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if err = t.Execute(res, "Register"); err != nil {
        log.Println(err)
        return
    }
}

func registerUserHandler(res http.ResponseWriter, req *http.Request) {
    req.ParseForm()
    username := req.FormValue("username")
    password := req.FormValue("password")

    salt := make([]byte, saltLen)
    if _, err := rand.Read(salt); err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    saltedHash, err := HashAndSaltPassword([]byte(password), salt)
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    user := User {
        Username: username,
        Salt: salt,
        SaltedHash: saltedHash,
    }

    users := db.Collection("users")
    users.InsertOne(context.Background(), user)
}

