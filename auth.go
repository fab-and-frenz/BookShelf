package main

import(
    "golang.org/x/crypto/scrypt"
    "context"
    "net/http"
    "html/template"
    "log"
    "time"
    "crypto/sha256"
    "crypto/subtle"
    "crypto/rand"
    "crypto/hmac"
    "encoding/base64"
    "encoding/json"
    "github.com/mongodb/mongo-go-driver/mongo/options"
    "github.com/mongodb/mongo-go-driver/bson"
)

var(
    hmacKey []byte
)

func init() {
    hmacKey := make([]byte, 32)
    if _, err := rand.Read(hmacKey); err != nil {
        log.Fatal(err)
    }
}

type JWT struct {
    Username   string `json:"username"`
    Expires  []byte   `json:"expires"`
}

type User struct {
    Username     string `bson:"username" json:"saltedhash"`
    Salt       []byte   `bson:"salt" json:"saltedhash"`
    SaltedHash []byte   `bson:"saltedhash" json:"saltedhash"`
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

    vals := struct{ Action, Name string }{ "/registeruser", "Register" }
    if err = t.Execute(res, vals); err != nil {
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

func loginPageHandler(res http.ResponseWriter, req *http.Request) {
    t, err := template.ParseFiles("html/auth.html")
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    vals := struct{ Action, Name string }{ "/loginuser", "Login" }
    if err = t.Execute(res, vals); err != nil {
        log.Println(err)
        return
    }
}

func loginUserHandler(res http.ResponseWriter, req *http.Request) {
    req.ParseForm()
    username := req.FormValue("username")
    password := req.FormValue("password")

    var user User
    err := usersCollection.FindOne(
        context.Background(),
        bson.D{{"username", username}},
        options.FindOne(),
    ).Decode(&user)
    if err != nil {
        res.WriteHeader(400)
        res.Write([]byte("Invalid username or password."))
        return
    }

    saltedHash, err := HashAndSaltPassword([]byte(password), user.Salt)
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    if subtle.ConstantTimeCompare(saltedHash, user.SaltedHash) == 1 {
        cookie := http.Cookie {
            Name: "session",
            Value: username,
            Secure: true,
            SameSite: 1,
            Path: "/",
        }

        http.SetCookie(res, &cookie)
        res.Write([]byte("login successful"))
    } else {
        res.WriteHeader(400)
        res.Write([]byte("Invalid username or password."))
    }
}

func createJWT(username string, hmacKey []byte) (string, error) {
    header := base64.URLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

    expireTime, err := time.Now().Add(time.Hour).MarshalText()
    if err != nil {
        return "", err
    }

    jsonPayload, err := json.Marshal(JWT{
        Username: username,
        Expires: expireTime,
    })
    if err != nil {
        return "", err
    }

    payload := base64.URLEncoding.EncodeToString(jsonPayload)
    jwt := header + "." + payload

    mac := hmac.New(sha256.New, hmacKey)
    mac.Write([]byte(jwt))
    tag := mac.Sum(nil)
    encodedTag := base64.URLEncoding.EncodeToString(tag)

    jwt = encodedTag + "." + jwt

    return string(jwt), nil
}

