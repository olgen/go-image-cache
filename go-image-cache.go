package main

import (
    "fmt"
    "log"
    "os"
    "time"
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

var (
  cacheControl = "max-age:290304000, public"
  cacheSince = time.Now().Format(http.TimeFormat)
	cacheUntil = time.Now().AddDate(60, 0, 0).Format(http.TimeFormat)
  client = initMemcacheClient()
  vBucket = (uint16)(0)
 )

func main(){
    http.HandleFunc("/", handleHttp)
    fmt.Println("listening...")
    err := http.ListenAndServe(port(), nil)
    if err != nil {
        panic(err)
    }
}


func initMemcacheClient() *memcached.Client {
    memcacheUrl := os.Getenv("MEMCACHED_URL")

    u, err := url.Parse(memcacheUrl)
    if err!= nil{
        log.Fatalf("Error parsing MEMCACHED_URL: %v", err)
    }
    protocol := u.Scheme
    host := u.Host


    client, err := memcached.Connect(protocol, host)
    if err != nil {
        log.Fatalf("Error connecting: %v", err)
    }

    log.Println("Connected to memcached host:", host)

    if u.User != nil {
        user := u.User.Username()
        pass, _ := u.User.Password()
        if user != "" {
            resp, err := client.Auth(user, pass)
            if err != nil {
                log.Fatalf("auth error: %v", err)
            }
            log.Printf("Auth response = %v", resp)
        }
    }
    return client
}

func handleHttp(w http.ResponseWriter, r *http.Request) {
    cacheKey := r.URL.String()
    responseData := loadFromCache(cacheKey)

    if responseData == nil {
        fmt.Println("Not found on Cache: ", cacheKey)
        responseData = loadFromOrigin(r.URL)
        cacheResponse(cacheKey, *responseData)
    }else{
        fmt.Println("Serving from cache: ", cacheKey)
    }

    serveResponse(*responseData, w)
    addCacheHeaders(w)
    addCorsHeaders(w)
}


const cacheLimit = 1024 * 1024 // memcached limit of 1MB
func cacheResponse(key string, data ResponseData) {
    if data.StatusCode != 200 {
        log.Printf("Not a success response: StatusCode=%v, not caching!", data.StatusCode)
        return
    }

    dump, err := serialize(data)
    if err != nil {
        fmt.Println("Serialization error:", err.Error())
        return
    }

    size := len(dump)
    if size >= cacheLimit {
        log.Printf("dump is too big: %v, not caching!", size)
        return
    }

    _, err = client.Set(vBucket, key, 0, 0, dump)
    if err != nil {
        log.Printf("Error caching key: %v", err)
    }
    log.Printf("Stored key=%v, size=%v to cache.", key, size)
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

func addCacheHeaders(w http.ResponseWriter) {
    w.Header().Set("Cache-Control", cacheControl)
		w.Header().Set("Last-Modified", cacheSince)
		w.Header().Set("Expires", cacheUntil)
}
func addCorsHeaders(w http.ResponseWriter){
    w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveResponse(data ResponseData, w http.ResponseWriter) {
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
