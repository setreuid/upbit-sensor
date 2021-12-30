package main

import (
    "context"
    "github.com/gin-contrib/gzip"
    "github.com/gin-gonic/gin"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

type Api struct{}

func (self *Api) Run() {
    router := self.setupRouter()

    srv := &http.Server{
        Addr:         "0.0.0.0:12502",
        Handler:      router,
        ReadTimeout:  30000 * time.Millisecond,
        WriteTimeout: 30000 * time.Millisecond,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    <-quit
    log.Println("Shutdown Server ...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        RUNNING = false
        log.Fatal("Server Shutdown: ", err)
    }
}

func (self *Api) setupRouter() *gin.Engine {
    r := gin.Default()
    r.Use(gzip.Gzip(gzip.BestCompression))
    r.Use(CORSMiddleware())

    r.GET("/status", ApiStatus)
    r.GET("/history", ApiHistory)

    return r
}

func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, Accept-Ranges, Range, bytes")
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, DELETE, POST")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

func ApiStatus(c *gin.Context) {
    result := map[string][]*StatusInfo{}

    for item := range status.Iter() {
        val := item.Value
        if val != nil {
            result[item.Key.(string)] = val.([]*StatusInfo)
        }
    }

    c.JSON(http.StatusOK, result)
}

func ApiHistory(c *gin.Context) {
    result := map[string]map[int32][]*StatusInfo{}

    c.JSON(http.StatusOK, result)
}
