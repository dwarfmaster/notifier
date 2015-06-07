package screens

import (
    "fmt"
    "github.com/BurntSushi/xgb"
    "github.com/BurntSushi/xgb/xinerama"
)

type Geometry struct {
    X, Y int16
    W, H uint16
}

type InvalidIdError uint
func (e InvalidIdError) Error() string {
    return fmt.Sprintf("Not a valid screen id : %v", uint(e))
}

var count uint32
var sizes []Geometry;

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
        sizes = append(sizes, Geometry{scr.XOrg, scr.YOrg, scr.Width, scr.Height})
    }
    return nil
}

func Count() uint32 {
    return count
}

func Focused() uint32 {
    /* TODO */
    return 0
}

func Geom(id uint32) (Geometry, error) {
    if id >= count {
        return Geometry{0, 0, 0, 0}, InvalidIdError(id)
    }
    return sizes[id], nil
}

