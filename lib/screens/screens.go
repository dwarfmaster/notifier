package screens

import (
    "fmt"
    "github.com/BurntSushi/xgb"
    "github.com/BurntSushi/xgb/xproto"
    "github.com/BurntSushi/xgb/xinerama"

    "github.com/lucas8/notifier/lib/types"
)

type InvalidIdError uint
func (e InvalidIdError) Error() string {
    return fmt.Sprintf("Not a valid screen id : %v", uint(e))
}

var count uint32
var sizes []types.Geometry;

func Load(c *xgb.Conn) error {
    err := xinerama.Init(c)
    if err != nil {
        return err
    }

    reply, err := xinerama.QueryScreens(c).Reply()
    if err != nil {
        return err
    }

    count = reply.Number
    for _, scr := range reply.ScreenInfo {
        sizes = append(sizes, types.Geometry{int32(scr.XOrg),  int32(scr.YOrg),
                                             int32(scr.Width), int32(scr.Height)})
    }
    return nil
}

func Count() uint32 {
    return count
}

func Focused(c *xgb.Conn) uint32 {
    incookie := xproto.GetInputFocus(c)
    rep, err := incookie.Reply()
    if err != nil {
        return 0
    }
    win := rep.Focus

    trcookie := xproto.TranslateCoordinates(c, win,
                                            xproto.Setup(c).DefaultScreen(c).Root,
                                            0, 0)
    att, err := trcookie.Reply()
    if err != nil {
        return 0
    }
    x,y := int32(att.DstX), int32(att.DstY)

    for i, size := range sizes {
        if size.X <= x && size.X + size.W >= x && size.Y <= y && size.Y + size.H >= y {
            return uint32(i)
        }
    }
    return 0
}

func Geom(id uint32) (types.Geometry, error) {
    if id >= count {
        return types.Geometry{0, 0, 0, 0}, InvalidIdError(id)
    }
    return sizes[id], nil
}

