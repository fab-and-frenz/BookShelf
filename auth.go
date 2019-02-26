package main

import(
    "golang.org/x/crypto/scrypt"
    "context"
    "net/http"
    "html/template"
    "log"
    "errors"
    "bytes"
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

    ErrNoSuchUser error = errors.New("No such user")
    ErrMalformedJWT error = errors.New("Malformed JWT")
    ErrInvalidMAC error = errors.New("Invalid MAC")
    ErrTokenExpired error = errors.New("Expired token")
)

func init() {
    hmacKey = make([]byte, 32)
    if _, err := rand.Read(hmacKey); err != nil {
        log.Fatal(err)
    }
}

type JWT struct {
    Username   string `json:"username"`
    Expires  []byte   `json:"expires"`
}

type User struct {
    Username     string `bson:"username"   json:"saltedhash"`
    Salt       []byte   `bson:"salt"       json:"saltedhash"`
    SaltedHash []byte   `bson:"saltedhash" json:"saltedhash"`

    Books      []Book   `bson:"books"      json:"books"`
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
        Books: []Book{},
    }

    users := db.Collection("users")
    users.InsertOne(context.Background(), user)

    http.Redirect(res, req, "/login", 302) 
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
        jwt, err := createJWT(username, hmacKey)
        if err != nil {
            res.WriteHeader(500)
            res.Write([]byte("Failed to login"))
        }

        cookie := http.Cookie {
            Name: "session",
            Value: jwt,
            Secure: true,
            SameSite: 1,
            Path: "/",
        }

        http.SetCookie(res, &cookie)
        http.Redirect(res, req, "/library", 302)
    } else {
        res.WriteHeader(400)
        res.Write([]byte("Invalid username or password."))
    }
}

func logoutHandler(res http.ResponseWriter, req *http.Request) {
        cookie := http.Cookie {
            Name: "session",
            Value: "-1",
            Secure: true,
            SameSite: 1,
            Path: "/",
        }

        http.SetCookie(res, &cookie)
        http.Redirect(res, req, "/login", 302)
}

// unwrapJWT checks if a JWT is valid (unexpired, and distributed by the server),
// and returns the payload as a JWT structure if it is valid.
func unwrapJWT(jwt, hmacKey []byte) (JWT, error) {
    var ret JWT

    separators := make([]int, 0, 2)
    for i := 0; i < len(jwt); i++ {
        if jwt[i] == '.' {
            separators = append(separators, i)
        }
    }

    if len(separators) != 2 {
        return ret, ErrMalformedJWT
    }

    payload := jwt[separators[0] + 1:separators[1]]
    mac := jwt[separators[1] + 1:]

    decodedMac := make([]byte, base64.URLEncoding.DecodedLen(len(mac)))
    if _, err := base64.URLEncoding.Decode(decodedMac, mac); err != nil {
        return ret, err
    }

    decodedMac = bytes.Trim(decodedMac, "\x00")

    if !validateMAC(jwt[:separators[1]], decodedMac, hmacKey) {
        return ret, ErrInvalidMAC
    }

    jsonPayload, err := base64.URLEncoding.DecodeString(string(payload))
    if err != nil {
        return ret, err
    }

    if err = json.Unmarshal(jsonPayload, &ret); err != nil {
        return ret, err
    }

    expirey := new(time.Time)
    if err = expirey.UnmarshalText(ret.Expires); err != nil {
        return ret, err
    }

    if time.Now().After(*expirey) {
        return ret, ErrTokenExpired
    }

    return ret, nil
}

// createJWT creates a JWT login token for the user specified by username,
// creating a MAC using the key specified by hmacKey.
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

    jwt = jwt + "." + encodedTag

    return string(jwt), nil
}

func checkSession(req *http.Request) (JWT, error) {
    cookie, err := req.Cookie("session")
    if err != nil {
        return JWT{}, err
    }

    jwt, err := unwrapJWT([]byte(cookie.Value), hmacKey)
    if err != nil {
        return JWT{}, err
    }

    return jwt, nil
}

func lookupUser(username string) (User, error) {
    var user User

    result := usersCollection.FindOne(
        context.Background(),
        bson.M{"username": username},
        options.FindOne(),
    )
    
    if result == nil {
        return user, ErrNoSuchUser
    }
    
    if err := result.Decode(&user); err != nil {
        return user, err
    }
    
    return user, nil
}

// validateMAC computes the MAC of message using the provided key,
// and compares it with messageMAC. If the computed MAC matches
// messageMAC, validateMAC will return true, otherwise it will return false.
func validateMAC(message, messageMAC, key []byte) bool {
    mac := hmac.New(sha256.New, key)
    mac.Write(message)
    expectedMAC := mac.Sum(nil)
    return subtle.ConstantTimeCompare(messageMAC, expectedMAC) == 1
}

