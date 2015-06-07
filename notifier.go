package main

import (
    "fmt"
    "github.com/BurntSushi/xgb"

    "github.com/lucas8/notifier/lib/config"
    "github.com/lucas8/notifier/lib/screens"
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
        c, err := xgb.NewConn()
        if err != nil {
            fmt.Printf("Error when connecting to x11 server : %v\n", err)
            return
        }
        conn = c
        defer conn.Close()
    }

    /* Loading screens configuration */
    {
        err := screens.Load(conn)
        if err != nil {
            fmt.Printf("Error while getting screens configuration : %s\n", err)
            return
        }
    }

    /* TODO */
}

