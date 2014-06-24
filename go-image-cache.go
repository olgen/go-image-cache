package main

import (
    "fmt"
    "os"
    "net/http"
    "net/url"
    "io/ioutil"
    "strings"
    "encoding/json"
    "github.com/bradfitz/gomemcache/memcache"
)

type ResponseData struct {
    ContentType    string
    Body []byte
    StatusCode int
}

var mc = memcache.New(memcacheServer())

func main(){
    http.HandleFunc("/", serveResponse)
    fmt.Println("listening...")
    err := http.ListenAndServe(port(), nil)
    if err != nil {
        panic(err)
    }
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
    dump, err := json.Marshal(data)
    if err != nil {
        fmt.Println("error:", err.Error())
    }
    mc.Set(&memcache.Item{Key: key, Value: dump})
}

func loadFromCache(key string) *ResponseData {
    var reloaded  ResponseData
    item, err := mc.Get(key)
    if err!=nil {
        fmt.Println("Error:", err.Error())
        return nil
    }
    dump := item.Value
    err1 := json.Unmarshal(dump, &reloaded)
    if err1 != nil {
        fmt.Println("error:", err1)
    }
    return &reloaded
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
func memcacheServer() string {
    url := os.Getenv("MEMCACHED_URL")
    if  url == "" {
        panic("No MEMCACHED_URL env-var given!")
    }
    return url

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
