package main

import (
  _ "os"
  "log"
  "flag"
  "bytes"
  "strings"
  "net/url"
  "net/http"
  "net/http/httputil"
  "io/ioutil"
  _ "encoding/json"
  "encoding/base64"
  "github.com/gorilla/mux"
)

type SSHProxy struct {
  url *url.URL
  proxy *httputil.ReverseProxy
}

type SSHTransport struct {
}

func SSH(target string) *SSHProxy {
  url, _ := url.Parse(target)
  return &SSHProxy{ url: url, proxy: httputil.NewSingleHostReverseProxy(url) }
}

func (p *SSHProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  p.proxy.Transport = &SSHTransport{}
  p.proxy.ServeHTTP(w, r)
}

func (t *SSHTransport) RoundTrip(request *http.Request) (*http.Response, error) {
  buf, err := ioutil.ReadAll(request.Body)
  if err != nil {
    return nil, err
  }
  rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
  rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))

  log.Println("Request body : ", rdr1)
  request.Body = rdr2


  response, err := http.DefaultTransport.RoundTrip(request)
  if err != nil {
    print("\n\ncame in error resp here", err)
      return nil, err //Server is not reachable. Server not working
  }

  body, err := httputil.DumpResponse(response, true)
  if err != nil {
    print("\n\nerror in dumb response")
    // copying the response body did not work
    return nil, err
  }

  log.Println("Response Body : ", string(body))
  return response, err
}

// Add data collection to front and back end

var ADMIN_USERNAME string
var ADMIN_PASSWORD string

var chainPath string = "/etc/letsencrypt/live/bubbajoe.us/fullchain.pem"

var keyPath string = "/etc/letsencrypt/live/bubbajoe.us/privkey.pem"

func main() {
  //port := flag.String("port", "80", "Port for application")
  flag.Parse()
  r := mux.NewRouter()
  // 404 Handler
  r.NotFoundHandler = http.HandlerFunc(HTTP404Handler)
  // File server
  fs := http.FileServer(http.Dir("static"))
  http.Handle("/", fs)
  //proxy := SSH("http://0.0.0.0:2222/")
  //http.Handle("/ssh/", proxy)
  http.HandleFunc("/admin", AdminHandler)
  //http.HandleFunc("/admin/setlogin", AdminLoginHandler)
  // RESTful
  http.HandleFunc("/get/repos", get_repos)
  //http.HandleFunc("/add/click", nil)
  //http.HandleFunc("/get/data", nil)
  go http.ListenAndServe("0.0.0.0:80", http.HandlerFunc(Redirect))//http.HandlerFunc(Redirect))
  http.ListenAndServeTLS("0.0.0.0:443", chainPath, keyPath, nil)
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

// RESTful
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

// Other
func Redirect(w http.ResponseWriter, req *http.Request) {
    // remove/add not default ports from req.Host
    target := "https://" + req.Host + req.URL.Path 
    if len(req.URL.RawQuery) > 0 {
        target += "?" + req.URL.RawQuery
    }
    log.Printf("redirect to: %s", target)
    http.Redirect(w, req, target,
            // see @andreiavrammsd comment: often 307 > 301
            http.StatusTemporaryRedirect)
}
