package main

import (
    "fmt"
    "log"
    "os"
    "net/http"
    "net/url"
    "io/ioutil"
    "strings"
    "encoding/json"
    "github.com/dustin/gomemcached/client"
)

type ResponseData struct {
    ContentType    string
    Body []byte
    StatusCode int
}

var client = initMemcacheClient()
var vBucket = (uint16)(0)

func main(){
    http.HandleFunc("/", serveResponse)
    fmt.Println("listening...")
    err := http.ListenAndServe(port(), nil)
    if err != nil {
        panic(err)
    }
}


func initMemcacheClient() *memcached.Client {
    host:= memcacheHost()
    var client, err = memcached.Connect("tcp", host)
    if err != nil {
        log.Fatalf("Error connecting: %v", err)
    }
    log.Println("Connected to MEMCACHED_HOST=", host)

    user := os.Getenv("MEMCACHED_USER")
    pass := os.Getenv("MEMCACHED_PASS")
    if user != "" {
        resp, err := client.Auth(user, pass)
        if err != nil {
            log.Fatalf("auth error: %v", err)
        }
        log.Printf("Auth response = %v", resp)
    }
    return client
}

func serveResponse(w http.ResponseWriter, r *http.Request) {
    cacheKey := r.URL.String()
    responseData := loadFromCache(cacheKey)

    if responseData == nil {
        fmt.Println("Not found on Cache: ", cacheKey)
        responseData = loadFromOrigin(r.URL)
        cacheResponse(cacheKey, *responseData)
    }else{
        fmt.Println("Serving from cache: ", cacheKey)
    }

    tunnelResponse(*responseData, w)
    addCorsHeaders(w)
}


func cacheResponse(key string, data ResponseData) {

    dump, err := serialize(data)
    if err != nil {
        fmt.Println("error:", err.Error())
        return
    }
    _, err = client.Set(vBucket, key, 0, 0, dump)
    if err != nil {
        log.Printf("Error setting key: %v", err)
    }
}

func loadFromCache(key string) *ResponseData {
    resp, err := client.Get(vBucket, key)
    if err != nil {
        log.Printf("Error retrieving key: %v", err)
        return nil
    }
    return deserialize(resp.Body)
}

func serialize(data ResponseData) ( []byte, error ){
    return json.Marshal(data)
}

func deserialize(dump []byte) *ResponseData {
    var data  ResponseData
    err1 := json.Unmarshal(dump, &data)
    if err1 != nil {
        fmt.Println("error:", err1)
        return nil
    }
    return &data
}

func addCorsHeaders(w http.ResponseWriter){
    w.Header().Set("Access-Control-Allow-Origin", "*")
}

func tunnelResponse(data ResponseData, w http.ResponseWriter) {
    w.WriteHeader(data.StatusCode)
    w.Header().Set("Content-Type", data.ContentType)
    w.Write(data.Body)
}

func loadFromOrigin(url *url.URL) *ResponseData {
    urlString := url.String()

    originUrl := strings.Replace(urlString, url.Host, originHost(), 1)
    fmt.Println("Loading from url=", originUrl )
    resp, err := http.Get(originUrl)
    if err != nil {
        fmt.Println("Error while loading: %s", err.Error())
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    resp.Body.Read(body)
    data := ResponseData{
        ContentType: resp.Header.Get("Content-Type"),
        Body: body,
        StatusCode: resp.StatusCode,
    }
    return &data
}

// Config values
func memcacheHost() string {
    url := os.Getenv("MEMCACHED_HOST")
    if  url == "" {
        panic("No MEMCACHED_HOST env-var given!")
    }
    return url
}

func originHost() string{
    origin := os.Getenv("ORIGIN")
    if origin == "" {
        panic("No ORIGIN env-var given!")
    }
    log.Println("Preparing to serve from ORIGIN=", origin)
    return origin
}

func port() string {
    port := os.Getenv("PORT")
    if port == "" {
        panic("No PORT env-var given!")
    }
    return ":" + port
}
