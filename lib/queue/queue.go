package queue

import (
    "fmt"
    "github.com/BurntSushi/xgb"

    "github.com/lucas8/notifier/lib/types"
    "github.com/lucas8/notifier/lib/window"
    "github.com/lucas8/notifier/lib/screens"
    "github.com/lucas8/notifier/lib/config"
)

const (
    grTopLeft = iota
    grTopRight
    grBottomLeft
    grBottomRight
)

type notif struct {
    onScreen bool
    screen int
    id uint32
    win *window.Window

    next *notif
    prev *notif
}

type Queue struct {
    conn *xgb.Conn
    /* The notifications for each screen */
    scrs []*notif
    mid uint32

    gravity int
    vertPad uint32
    horiPad uint32
    space   uint32
    initPad uint32
}

func Open(c *xgb.Conn) (*Queue, error) {
    var q Queue
    q.conn = c
    q.scrs = make([]*notif, screens.Count())

    q.gravity = grTopRight
    if gr, err := config.String("global.gravity"); err == nil {
        switch gr {
        case "top_left":     q.gravity = grTopLeft
        case "top_right":    q.gravity = grTopRight
        case "bottom_right": q.gravity = grBottomRight
        case "bottom_left":  q.gravity = grBottomLeft
        }
    }

    q.vertPad = 15
    if nb, err := config.Int("global.padding.vert"); err == nil {
        q.vertPad = uint32(nb)
    }

    q.horiPad = 15
    if nb, err := config.Int("global.padding.hori"); err == nil {
        q.horiPad = uint32(nb)
    }

    q.space = 15
    if nb, err := config.Int("global.padding.space"); err == nil {
        q.space = uint32(nb)
    }

    q.mid = 0
    return &q, nil
}

type ClosedChannelError struct {}
func (e ClosedChannelError) Error() string {
    return "closed order channel"
}

func (q *Queue) closeAllNotif() {
    for i, not := range q.scrs {
        for not != nil {
            not.win.Close()
            not = not.next
        }
        q.scrs[i] = nil
    }
}

func (q *Queue) updatePos(scr int) {
    not := q.scrs[scr]
    y := int32(q.space)
    g, _ := screens.Geom(uint32(scr))
    for not != nil && y < g.H {
        gn := not.win.Geom()
        yn := y + int32(q.space) + gn.H
        if yn <= g.H {
            ym := y
            xm := int32(0)
            switch q.gravity {
            case grTopRight:
                xm = g.W - gn.W - int32(q.vertPad)
            case grTopLeft:
                xm = int32(q.vertPad)
            case grBottomLeft:
                xm = int32(q.vertPad)
                ym = g.H - y - gn.H
            case grBottomRight:
                xm = g.W - gn.W - int32(q.vertPad)
                ym = g.H - y - gn.H
            }
            not.win.Move(uint32(xm), uint32(ym))
            if !not.onScreen {
                not.win.Map()
                not.onScreen = true
            }
        }
        y = yn
    }
}

func (q *Queue) closeNotif(n *notif) {
    if n.prev != nil {
        n.prev.next = n.next
    }
    if n.next != nil {
        n.next.prev = n.prev
    }
    if q.scrs[n.screen] == n {
        q.scrs[n.screen] = n.next
    }
    q.updatePos(n.screen)
}

func (q *Queue) findNotifById(id uint32) *notif {
    for _, not := range q.scrs {
        for not != nil {
            if not.id == id {
                return not
            }
            not = not.next
        }
    }
    return nil
}

func (q *Queue) openNotif(lvl, txt string, time uint32) {
    /* TODO handle time */
    var not notif
    scr := screens.Focused(q.conn)
    not.onScreen = false
    not.screen = int(scr)
    not.id = q.mid
    q.mid++
    not.win, _ = window.Open(q.conn, lvl, "Notification", txt)
    not.next = nil

    if q.scrs[scr] == nil {
        not.prev = nil
        q.scrs[scr] = &not
    } else {
        p := q.scrs[scr]
        for p.next != nil {
            p = p.next
        }
        p.next = &not
        not.prev = p
    }
}

func (q *Queue) redraw() {
    for _, not := range q.scrs {
        for not != nil {
            not.win.Redraw()
            not = not.next
        }
    }
}

/* Will run processing incoming orders from c until it is killed (return nil)
 * or a fatal error happen (return it)
 */
func (q *Queue) Run(c chan types.Order) error {
    o, ok := <-c
    for ok {
        switch ord := o.(type) {
        case types.KillOrder:
            return nil
        case types.CloseOrder:
            if ord.All {
                q.closeAllNotif()
            } else if ord.Top {
                scr := screens.Focused(q.conn)
                q.closeNotif(q.scrs[scr])
            } else {
                q.closeNotif(q.findNotifById(ord.Id))
            }
        case types.NotifOrder:
            q.openNotif(ord.Level, ord.Text, ord.Time)
        case types.RedrawOrder:
            q.redraw()
        }
        o, ok = <-c
    }
    return ClosedChannelError{}
}

