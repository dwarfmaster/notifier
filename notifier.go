package main

import (
    "fmt"
    "strings"
    "strconv"
    "github.com/BurntSushi/xgb"
    "github.com/BurntSushi/xgb/xproto"

    "github.com/lucas8/notifier/lib/config"
    "github.com/lucas8/notifier/lib/screens"
    "github.com/lucas8/notifier/lib/window"
    "github.com/lucas8/notifier/lib/fifo"
    "github.com/lucas8/notifier/lib/queue"
    "github.com/lucas8/notifier/lib/types"
)

type KillCommand struct {}
func (c *KillCommand) Validate(str string) bool {
    return str == "kill" || str == "end"
}
func (c *KillCommand) Get() types.Order {
    return types.KillOrder {}
}

type RedrawCommand struct {}
func (c *RedrawCommand) Validate(str string) bool {
    return str == "redraw"
}
func (c *RedrawCommand) Get() types.Order {
    return types.RedrawOrder {}
}

type CloseCommand types.CloseOrder
func (c *CloseCommand) Validate(str string) bool {
    if str == "close" {
        c.All = false
        c.Top = true
        return true
    } else if str == "close_all" {
        c.All = true
        c.Top = false
        return true
    }
    return false
}
func (c *CloseCommand) Get() types.Order {
    return types.CloseOrder(*c)
}

type NotifCommand types.NotifOrder
func (c *NotifCommand) Validate(str string) bool {
    parts := strings.Fields(str)
    if len(parts) != 4  || parts[0] != "notif" {
        return false
    }
    t, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return false
    }
    c.Time  = uint32(t)
    c.Level = parts[2]
    c.Text  = parts[3]
    return true
}
func (c *NotifCommand) Get() types.Order {
    return types.NotifOrder(*c)
}

func main() {
    /* Loading config */
    if err := config.Load(config.ConfigPath()); err != nil {
        fmt.Printf("Error when loading config : %v\n", err)
        return
    }

    /* Opening the connection */
    var conn *xgb.Conn
    if c, err := xgb.NewConn(); err != nil {
        fmt.Printf("Error when connecting to x11 server : %v\n", err)
        return
    } else {
        conn = c
    }
    defer conn.Close()

    /* Loading screens configuration */
    if err := screens.Load(conn); err != nil {
        fmt.Printf("Error while getting screens configuration : %v\n", err)
        return
    }

    /* Loading window manager */
    if err := window.Load(conn); err != nil {
        fmt.Printf("Error while loading window manager : %v\n", err)
        return
    }

    /* Opening the fifo */
    var pipe *fifo.Fifo
    if p, err := fifo.Open(); err != nil {
        fmt.Printf("Error while opening the fifo : %s\n", err)
        return
    } else {
        pipe = p
    }
    defer pipe.Close()
    cmds := [...]fifo.Command {
        &KillCommand {},
        &RedrawCommand {},
        &CloseCommand {false, false, 0},
        &NotifCommand {0, "", ""},
    }
    for _, cmd := range cmds {
        pipe.AddCmd(cmd)
    }

    /* Opening the queue */
    var notifs *queue.Queue
    if q, err := queue.Open(conn); err != nil {
        fmt.Printf("Error while opening the queue : %s\n", err)
        return
    } else {
        notifs = q
    }

    /* Main loop */
    orders := make(chan types.Order, 10)
    go xloop(conn, orders)
    go pipe.ReadOrders(orders)
    notifs.Run(orders)
}

func xloop(conn *xgb.Conn, c chan types.Order) {
    for {
        ev, xerr := conn.WaitForEvent()
        if ev == nil && xerr == nil {
            c <- types.KillOrder {}
        }

        if ev != nil {
            switch ev.(type) {
            case xproto.ExposeEvent:
                c <- types.RedrawOrder {}
            }
        }
    }
}

