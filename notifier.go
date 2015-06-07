package main

import (
    "fmt"
    "os"
    "github.com/lucas8/notifier/lib/config"
    "code.google.com/p/x-go-binding/xgb"
)

func main() {
    /* Loading config */
    {
        path := config.ConfigPath()
        err := config.Load(path)
        if err != nil {
            fmt.Printf("Error when loading config : %v\n", err)
            return
        }
    }

    /* Opening the connection */
    var conn *xgb.Conn
    {
        c, err := xgb.Dial(os.Getenv("DISPLAY"))
        if err != nil {
            fmt.Printf("Error when connecting to x11 server : %v\n", err)
            return
        }
        conn = c
        defer conn.Close()
    }

    /* TODO */
}

