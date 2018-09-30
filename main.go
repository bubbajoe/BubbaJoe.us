package main

import (
  _ "os"
  "log"
  "flag"
  "strings"
  _ "net/url"
  "net/http"
  "io/ioutil"
  _ "encoding/json"
  "encoding/base64"
  "github.com/gorilla/mux"
)

// Add data collection to front and back end

var ADMIN_USERNAME string
var ADMIN_PASSWORD string

func main() {
  port := flag.String("port", "3000", "Port for application")
  flag.Parse()
  r := mux.NewRouter()
  // 404 Handler
  r.NotFoundHandler = http.HandlerFunc(HTTP404Handler)
  // File server
  fs := http.FileServer(http.Dir("static"))
  http.Handle("/", fs)
  http.HandleFunc("/admin", AdminHandler)
  //http.HandleFunc("/admin/setlogin", AdminLoginHandler)
  // RESTful
  http.HandleFunc("/get/repos", get_repos)
  //http.HandleFunc("/add/click", nil)
  //http.HandleFunc("/get/data", nil)
  log.Println("http://localhost:"+*port)
  http.ListenAndServe(":"+*port, nil)
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
  if auth, err := Authorized(w,r); !auth {
    if err != nil {
      //http.Error(w, err, 401)
      log.Println(err)
    } else {
      http.Error(w, "Not Authorized", 401)
    }
    return
  }  

  w.Write([]byte("Admin Section of BubbaJoe.us coming soon."))
}

func Authorized(w http.ResponseWriter, r *http.Request) (bool, error) {
  u := ADMIN_USERNAME
  p := ADMIN_PASSWORD

  w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
  s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
  
  if len(s) != 2 {
    return false, nil
  }

  b, err := base64.StdEncoding.DecodeString(s[1])
  if err != nil {
    return false, err
  }

  pair := strings.SplitN(string(b), ":", 2)
  if len(pair) != 2 {
    return false, nil
  }

  if pair[0] != u || pair[1] != p {
    return false, nil
  }
  return true, nil
}

func HTTP404Handler(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusNotFound)
}

func get_repos(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type","application/json")
  rs, err := http.Get("https://api.github.com/users/bubbajoe/repos?sort=updated")
  if err != nil {
      w.WriteHeader(http.StatusNotFound)
  }
  defer rs.Body.Close()

  body, err := ioutil.ReadAll(rs.Body)
  if err != nil {
      w.WriteHeader(http.StatusNotFound)
  }

  w.Write(body)
  //log.Println(t)
}