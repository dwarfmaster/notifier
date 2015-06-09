package window

import (
    "fmt"
    "strings"
    "github.com/BurntSushi/xgb"
    "github.com/BurntSushi/xgb/xproto"

    "github.com/lucas8/notifier/lib/config"
)

const defaultFont = "-*-terminal-medium-r-*-*-14-*-*-*-*-*-iso8859-*"

type color struct {
    r, g, b uint8
}

type gcontextdata struct {
    fg, bg, bc uint32
    width uint32
    font uint32
}
var defaultgc gcontextdata

type gcontext struct {
    fg, bg, bc xproto.Gcontext
    font xproto.Font
    width uint32
    fontHeight uint32
}
var ctxs map[string]*gcontext

type Window struct {
    next *Window
    prev *Window
}
var root *Window

type InvalidConfig string
func (e InvalidConfig) Error() string {
    return fmt.Sprintf("Bad confguration : %v", string(e))
}

func charValue(c byte) uint8 {
    if c >= 'A' && c <= 'F' {
        return 10 + c - 'A'
    } else if c >= '0' && c <= '9' {
        return c - '0'
    }
    return 0
}

func readColor(str string) color {
    clr := color{0, 0, 0}

    switch len(str) {
    case 2: /* Format '#G' */
        if str[0] != '#' {
            break
        }
        str = str[1:]
        fallthrough
    case 1: /* Format 'G' */
        clr.r = 16 * charValue(str[0])
        clr.g = clr.r
        clr.b = clr.r
    case 4: /* Format '#RGB' */
        if str[0] != '#' {
            break
        }
        str = str[1:]
        fallthrough
    case 3: /* Format 'RGB' */
        clr.r = 16 * charValue(str[0])
        clr.g = 16 * charValue(str[1])
        clr.b = 16 * charValue(str[2])
    case 7: /* Format '#RRGGBB' */
        if str[0] != '#' {
            break
        }
        str = str[1:]
        fallthrough
    case 6: /* Format 'RRGGBB' */
        clr.r = 16 * charValue(str[0]) + charValue(str[1])
        clr.g = 16 * charValue(str[2]) + charValue(str[3])
        clr.b = 16 * charValue(str[4]) + charValue(str[5])
    }
    return clr
}

func openFont(c *xgb.Conn, font string) (xproto.Font, error) {
    id, err := xproto.NewFontId(c)
    if err != nil {
        return 0, err
    }
    err = xproto.OpenFontChecked(c, id, uint16(len(font)), font).Check()
    return id, err
}

func openColor(c *xgb.Conn, scr *xproto.ScreenInfo, col color) (uint32, error) {
    cmap := scr.DefaultColormap
    rep, err := xproto.AllocColor(c, cmap,
                uint16(col.r) * 255, uint16(col.g) * 255, uint16(col.b) * 255).Reply()
    if err != nil {
        return 0, err
    }
    return rep.Pixel, nil
}

func loadDefaultGC(c *xgb.Conn, scr *xproto.ScreenInfo) {
    var font string
    if config.Has("global.gc.font") {
        font, _ = config.String("global.gc.font")
    } else {
        font = defaultFont
    }
    fnt, _ := openFont(c, font)
    defaultgc.font = uint32(fnt)

    if config.Has("global.gc.width") {
        wd, _ := config.Int("global.gc.width")
        defaultgc.width = uint32(wd)
    } else {
        defaultgc.width = 5
    }

    var cl color
    if config.Has("global.gc.fg") {
        str, _ := config.String("global.gc.fg")
        cl = readColor(str)
    } else {
        cl = color{255, 255, 255}
    }
    defaultgc.fg, _ = openColor(c, scr, cl)

    if config.Has("global.gc.bg") {
        str, _ := config.String("global.gc.bg")
        cl = readColor(str)
    } else {
        cl = color{0, 0, 0}
    }
    defaultgc.bg, _ = openColor(c, scr, cl)

    if config.Has("global.gc.bc") {
        str, _ := config.String("global.gc.bc")
        cl = readColor(str)
    } else {
        cl = color{255, 255, 255}
    }
    defaultgc.bc, _ = openColor(c, scr, cl)
}

func defaultGCValues(c *xgb.Conn) []uint32 {
    values := make([]uint32, 4)
    values[0] = defaultgc.fg
    values[1] = defaultgc.bg
    values[2] = defaultgc.width
    values[3] = defaultgc.font
    return values
}

func loadGC(name string, c *xgb.Conn, scr *xproto.ScreenInfo) error {
    var gc gcontext

    /* Foreground GC */
    id, err := xproto.NewGcontextId(c)
    if err != nil {
        return err
    }
    var mask uint32 = xproto.GcForeground | xproto.GcBackground |
                      xproto.GcLineWidth  | xproto.GcFont
    values := defaultGCValues(c)
    {
        cl, e := config.String(name + ".gc.fg")
        if e == nil {
            values[0], _ = openColor(c, scr, readColor(cl))
        }
        cl, e = config.String(name + ".gc.bg")
        if e == nil {
            values[1], _ = openColor(c, scr, readColor(cl))
        }
        wd, e := config.Int(name + ".gc.width")
        if e == nil {
            values[2] = uint32(wd)
        }
        cl, e = config.String(name + ".gc.font")
        if e == nil {
            fn, _ := openFont(c, cl)
            values[3] = uint32(fn)
        }
    }
    err = xproto.CreateGCChecked(c, id, xproto.Drawable(scr.Root), mask, values).Check()
    if err != nil {
        return err
    }
    gc.fg = id
    gc.font = xproto.Font(values[3])

    /* Query the font height */
    {
        rep, _ := xproto.QueryFont(c, xproto.Fontable(gc.font)).Reply()
        gc.fontHeight = uint32(rep.FontAscent) + uint32(rep.FontDescent)
    }

    /* Background GC */
    id, err = xproto.NewGcontextId(c)
    if err != nil {
        return err
    }
    mask = xproto.GcForeground | xproto.GcBackground | xproto.GcLineWidth
    values[0], values[1] = values[1], values[0]
    err = xproto.CreateGCChecked(c, id, xproto.Drawable(scr.Root), mask, values).Check()
    if err != nil {
        return err
    }
    gc.bg = id

    /* Border GC */
    id, err = xproto.NewGcontextId(c)
    if err != nil {
        return err
    }
    {
        cl, e := config.String(name + ".gc.bc")
        if e != nil {
            values[0], _ = openColor(c, scr, readColor(cl))
        }
    }
    values[1] = values[0]
    err = xproto.CreateGCChecked(c, id, xproto.Drawable(scr.Root), mask, values).Check()
    if err != nil {
        return err
    }
    gc.bc = id

    ctxs[name] = &gc
    return nil
}

func loadGCS(c *xgb.Conn, scr *xproto.ScreenInfo) error {
    if !config.Has("global.list") {
        return InvalidConfig("no global.list")
    }
    loadDefaultGC(c, scr)

    list, _ := config.String("global.list")
    entries := strings.Split(list, ",")
    for _, entry := range entries {
        err := loadGC(entry, c, scr)
        if err != nil {
            /* TODO Clean previously loaded gcs */
            return err
        }
    }
    return nil
}

func Load(c *xgb.Conn) error {
    ctxs = make(map[string]*gcontext)
    scr := xproto.Setup(c).DefaultScreen(c)
    err := loadGCS(c, scr)
    if err != nil {
        return err
    }

    /* TODO */
    return nil
}

func Has(name string) bool {
    _, has := ctxs[name]
    return has
}

func (w *Window) Close() {
    /* TODO */
}

func Open(c *xgb.Conn, gc, title, text string) (*Window, error) {
    /* TODO */
    return nil, nil
}

