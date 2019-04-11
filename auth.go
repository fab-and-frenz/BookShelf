package main

import(
    jwt "github.com/dgrijalva/jwt-go"
    "github.com/go-chi/jwtauth"
    "golang.org/x/crypto/scrypt"
    "context"
    "net/http"
    "html/template"
    "log"
    "errors"
    "time"
    "crypto/subtle"
    "crypto/rand"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/mongodb/mongo-go-driver/bson"
)

var(
    // The hmac key used to sign and verify JWTs
    hmacKey []byte

    tokenAuth *jwtauth.JWTAuth 

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

    tokenAuth = jwtauth.New("HS256", hmacKey, hmacKey)
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
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
            "username": username,
            "exp": time.Now().Add(30 * time.Minute).Unix(),
        })

        tokenString, err := token.SignedString(hmacKey)
        if err != nil {
            log.Println(err)
            res.WriteHeader(500)
            return
        }

        // Store the JWT as a cookie
        cookie := http.Cookie {
            Name: "jwt",
            Value: tokenString,
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
            Name: "jwt",
            Value: "-1",
            Secure: true,
            SameSite: 1,
            Path: "/",
        }

        http.SetCookie(res, &cookie)
        http.Redirect(res, req, "/login", 302)
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

