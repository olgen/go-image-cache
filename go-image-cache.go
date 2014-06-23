package main

import (
    "fmt"
    "net/http"
    "net/url"
    "os"
    "io"
    "strings"
)

func main(){
    http.HandleFunc("/", serveFromOrigin)
    fmt.Println("listening...")
    err := http.ListenAndServe(port(), nil)
    if err != nil {
        panic(err)
    }
}

func serveFromOrigin(w http.ResponseWriter, r *http.Request) {
    resp := loadFromOrigin(r.URL)
    tunnelResponse(resp, w)
    addCorsHeaders(w)
}

func addCorsHeaders(w http.ResponseWriter){
    w.Header().Set("Access-Control-Allow-Origin", "*")
}

func tunnelResponse(resp *http.Response, w http.ResponseWriter) {
    w.WriteHeader(resp.StatusCode)

    w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
    defer resp.Body.Close()
    io.Copy(w, resp.Body)
}

func loadFromOrigin(url *url.URL) *http.Response {
    urlString := url.String()

    originUrl := strings.Replace(urlString, url.Host, originHost(), 1)
    fmt.Println("Loading from url=", originUrl )
    resp, err := http.Get(originUrl)
    if err != nil {
        fmt.Println("Error while loading: %s", err.Error())
        return nil
    }
    return resp
}

func originHost() string{
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        panic("No ORIGIN env-var given!")
    }
    return origin
}

func port() string {
    port := os.Getenv("PORT")
    if port == "" {
        panic("No PORT env-var given!")
    }
    return ":" + port
}
