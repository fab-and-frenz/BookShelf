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
    // The hmac key used to sign JWTs
    hmacKey []byte

    ErrNoSuchUser error = errors.New("No such user")
    ErrMalformedJWT error = errors.New("Malformed JWT")
    ErrInvalidMAC error = errors.New("Invalid MAC")
    ErrTokenExpired error = errors.New("Expired token")
)

func init() {
    // Create a random HMAC key
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

// HashAndSaltPassword hashes password with the given salt, using SCrypt
func HashAndSaltPassword(password, salt []byte) ([]byte, error) {
    return scrypt.Key(password, salt, scryptN, scryptR, scryptP, hashLen)
}

// registerPageHandler shows the html frontend for the register page
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

// registerUserHandler handles requests to register a user with the system.
// This requests happens when someone submits their registration details from the register page.
func registerUserHandler(res http.ResponseWriter, req *http.Request) {
    req.ParseForm()

    // Get the submitted username and password
    username := req.FormValue("username")
    password := req.FormValue("password")

    // Create a random salt for the user
    salt := make([]byte, saltLen)
    if _, err := rand.Read(salt); err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    // Hash the password
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

    // Update the database with the new user
    users := db.Collection("users")
    users.InsertOne(context.Background(), user)

    // Redirect to the login page after registration
    http.Redirect(res, req, "/login", 302) 
}

// loginPageHandler shows the html for the login page
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

// loginUserHandler handles a request to log-in.
// This is called when someone sends their credentials from the login page.
func loginUserHandler(res http.ResponseWriter, req *http.Request) {
    req.ParseForm()

    // Get the submitted username and password
    username := req.FormValue("username")
    password := req.FormValue("password")

    // Retrieve that user from the database
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

    // Hash the password supplied by the user
    saltedHash, err := HashAndSaltPassword([]byte(password), user.Salt)
    if err != nil {
        res.WriteHeader(500)
        log.Println(err)
        return
    }

    // If the computed and stored hashes match, log the user in
    if subtle.ConstantTimeCompare(saltedHash, user.SaltedHash) == 1 {
        // Create a JWT for the user
        jwt, err := createJWT(username, hmacKey)
        if err != nil {
            res.WriteHeader(500)
            res.Write([]byte("Failed to login"))
        }

        // Store the JWT as a cookie
        cookie := http.Cookie {
            Name: "session",
            Value: jwt,
            Secure: true,
            SameSite: 1,
            Path: "/",
        }

        http.SetCookie(res, &cookie)

        // Direct the user to the library page after they have logged in
        http.Redirect(res, req, "/library", 302)
    } else {
        res.WriteHeader(400)
        res.Write([]byte("Invalid username or password."))
    }
}

// logoutHandler invalidates a cookie that has been set, by setting it to "-1"
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

    // Find the locations of the separators in the JWT
    separators := make([]int, 0, 2)
    for i := 0; i < len(jwt); i++ {
        if jwt[i] == '.' {
            separators = append(separators, i)
        }
    }

    if len(separators) != 2 {
        return ret, ErrMalformedJWT
    }

    // Extract the payload and MAC
    payload := jwt[separators[0] + 1:separators[1]]
    mac := jwt[separators[1] + 1:]

    // Decode the Base64-encoded MAC
    decodedMac := make([]byte, base64.URLEncoding.DecodedLen(len(mac)))
    if _, err := base64.URLEncoding.Decode(decodedMac, mac); err != nil {
        return ret, err
    }

    // Remove any null bytes
    decodedMac = bytes.Trim(decodedMac, "\x00")

    // Verify the MAC tag
    if !validateMAC(jwt[:separators[1]], decodedMac, hmacKey) {
        return ret, ErrInvalidMAC
    }

    // Decode the Base64  payload
    jsonPayload, err := base64.URLEncoding.DecodeString(string(payload))
    if err != nil {
        return ret, err
    }

    // Decode the JSON into a JWT structure
    if err = json.Unmarshal(jsonPayload, &ret); err != nil {
        return ret, err
    }

    // Verify that the JWT is still valid
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
    // Create the header
    header := base64.URLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

    // Set the token to expire in one hour
    expireTime, err := time.Now().Add(time.Hour).MarshalText()
    if err != nil {
        return "", err
    }

    // Convert the JWT into a JSON string
    jsonPayload, err := json.Marshal(JWT{
        Username: username,
        Expires: expireTime,
    })
    if err != nil {
        return "", err
    }

    // Concatenate the header and payload
    payload := base64.URLEncoding.EncodeToString(jsonPayload)
    jwt := header + "." + payload

    // Compute the HMAC tag for the token
    mac := hmac.New(sha256.New, hmacKey)
    mac.Write([]byte(jwt))
    tag := mac.Sum(nil)
    encodedTag := base64.URLEncoding.EncodeToString(tag)

    // Concatenate the HMAC tag to the end of the JWT
    jwt = jwt + "." + encodedTag

    return string(jwt), nil
}

// checkSession checks if a user is logged in or not.
// If the user is logged in, the user's JWT is returned.
// If the user is not logged in, an error is returned, and JWT will be nil
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

// lookupUser checks if a user exists in the database, and returns
// that user's record if he does.
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

