package main

import(
    "context"
    "net/http"
    "html/template"
    "log"
    "crypto/subtle"
    "github.com/mongodb/mongo-go-driver/mongo/options"
    "github.com/mongodb/mongo-go-driver/bson"
)

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
        res.Write([]byte("Login successful."))
    } else {
        res.WriteHeader(400)
        res.Write([]byte("Invalid username or password."))
    }
}

