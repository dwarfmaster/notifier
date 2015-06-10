package window
/* TODO EWMH support */
/* TODO ICCCM support */

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
    border uint32
    font uint32
}
var defaultgc gcontextdata

type gcontext struct {
    fg, bg, bc xproto.Gcontext
    font xproto.Font
    width uint32
    border uint32
    fontHeight uint32
    fontUp uint32
}
var ctxs map[string]*gcontext

type Window struct {
    id xproto.Window
    conn *xgb.Conn
    lines []string
    gc *gcontext
    height uint32
}

type InvalidConfig string
func (e InvalidConfig) Error() string {
    return fmt.Sprintf("Bad configuration : %v", string(e))
}

func charValue(c byte) uint8 {
    if c >= 'A' && c <= 'F' {
        return 10 + c - 'A'
    } else if c >= 'a' && c <= 'f' {
        return 10 + c - 'a'
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

func loadDefaultGC(c *xgb.Conn, scr *xproto.ScreenInfo) error {
    var font string
    if config.Has("global.gc.font") {
        font, _ = config.String("global.gc.font")
    } else {
        font = defaultFont
    }
    fnt, err := openFont(c, font)
    if err != nil {
        return err
    }
    defaultgc.font = uint32(fnt)

    if config.Has("global.width") {
        wd, _ := config.Int("global.width")
        defaultgc.width = uint32(wd)
    } else {
        defaultgc.width = 500
    }

    if config.Has("global.gc.width") {
        wd, _ := config.Int("global.gc.width")
        defaultgc.border = uint32(wd)
    } else {
        defaultgc.border = 5
    }

    var cl color
    if config.Has("global.gc.fg") {
        str, _ := config.String("global.gc.fg")
        cl = readColor(str)
    } else {
        cl = color{255, 255, 255}
    }
    defaultgc.fg, err = openColor(c, scr, cl)
    if err != nil {
        return err
    }

    if config.Has("global.gc.bg") {
        str, _ := config.String("global.gc.bg")
        cl = readColor(str)
    } else {
        cl = color{0, 0, 0}
    }
    defaultgc.bg, err = openColor(c, scr, cl)
    if err != nil {
        return err
    }

    if config.Has("global.gc.bc") {
        str, _ := config.String("global.gc.bc")
        cl = readColor(str)
    } else {
        cl = color{255, 255, 255}
    }
    defaultgc.bc, err = openColor(c, scr, cl)
    if err != nil {
        return err
    }
    return nil
}

func defaultGCValues(c *xgb.Conn) []uint32 {
    values := make([]uint32, 4)
    values[0] = defaultgc.fg
    values[1] = defaultgc.bg
    values[2] = defaultgc.border
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
            values[0], e = openColor(c, scr, readColor(cl))
            if e != nil {
                return e
            }
        }
        cl, e = config.String(name + ".gc.bg")
        if e == nil {
            values[1], e = openColor(c, scr, readColor(cl))
            if e != nil {
                return e
            }
        }
        wd, e := config.Int(name + ".gc.width")
        if e == nil {
            values[2] = uint32(wd)
        }
        cl, e = config.String(name + ".gc.font")
        if e == nil {
            fn, e := openFont(c, cl)
            if e != nil {
                return e
            }
            values[3] = uint32(fn)
        }
    }
    err = xproto.CreateGCChecked(c, id, xproto.Drawable(scr.Root), mask, values).Check()
    if err != nil {
        return err
    }
    gc.fg = id
    gc.font = xproto.Font(values[3])
    gc.border = values[2]

    /* Query the font height */
    {
        rep, e := xproto.QueryFont(c, xproto.Fontable(gc.font)).Reply()
        if e != nil {
            return e
        }
        gc.fontHeight = uint32(rep.FontAscent) + uint32(rep.FontDescent)
        gc.fontUp = uint32(rep.FontAscent)
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
        if e == nil {
            values[0], e = openColor(c, scr, readColor(cl))
            if e != nil {
                return e
            }
        } else {
            values[0] = defaultgc.bc
        }
    }
    values[1] = values[0]
    err = xproto.CreateGCChecked(c, id, xproto.Drawable(scr.Root), mask, values).Check()
    if err != nil {
        return err
    }
    gc.bc = id

    /* Width */
    {
        wd, e := config.Int(name + ".width")
        if e == nil {
            gc.width = uint32(wd)
        } else {
            gc.width = defaultgc.width
        }
    }

    ctxs[name] = &gc
    return nil
}

func loadGCS(c *xgb.Conn, scr *xproto.ScreenInfo) error {
    if !config.Has("global.list") {
        return InvalidConfig("no global.list")
    }
    err := loadDefaultGC(c, scr)
    if err != nil {
        return err
    }

    list, _ := config.String("global.list")
    entries := strings.Split(list, ",")
    for _, entry := range entries {
        err = loadGC(entry, c, scr)
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
    return nil
}

func Has(name string) bool {
    _, has := ctxs[name]
    return has
}

func (w *Window) Close() {
    xproto.DestroyWindow(w.conn, w.id)
}

func (w *Window) Move(x uint32, y uint32) {
    var mask uint16 = xproto.ConfigWindowX | xproto.ConfigWindowY
    values := make([]uint32, 2)
    values[0] = x
    values[1] = y
    xproto.ConfigureWindow(w.conn, w.id, mask, values)
}

type _word struct {
    wd string
    length uint32
    cookie xproto.QueryTextExtentsCookie
}

func prepString(str string) ([]xproto.Char2b, uint16) {
    ln := uint16(len(str))
    var ch2b []xproto.Char2b = make([]xproto.Char2b, ln)
    for i := uint16(0); i < ln; i++ {
        ch2b[i].Byte1 = str[i]
        ch2b[i].Byte2 = 0
    }
    return ch2b, ln
}

func cutLines(c *xgb.Conn, w uint32, font xproto.Font, text string) []string {
    wds := strings.Split(text, " ")
    var words []_word = make([]_word, len(wds))
    for i, word := range wds {
        words[i].wd     = word + " "
        words[i].length = 0
        ch2dstr, ln := prepString(words[i].wd)
        words[i].cookie = xproto.QueryTextExtents(c, xproto.Fontable(font), ch2dstr, ln)
    }

    var lines []string = make([]string, 0, 10)
    var line string    = ""
    var ln uint32      = 0
    for i := 0; i < len(words); i++ {
        rep, _ := words[i].cookie.Reply()
        words[i].length = uint32(rep.OverallWidth)
        ln += words[i].length
        if ln >= w {
            lines = append(lines, line)
            line = ""
            ln = words[i].length
        }
        line += words[i].wd
    }
    if ln != 0 {
        lines = append(lines, line)
    }

    return lines
}

type BadContextError string
func (e BadContextError) Error() string {
    return fmt.Sprintf("Can't open notification with inexistant context : %s", string(e))
}

func Open(c *xgb.Conn, ctx, title, text string) (*Window, error) {
    gc, ok := ctxs[ctx]
    if !ok {
        return nil, BadContextError(ctx)
    }

    scr := xproto.Setup(c).DefaultScreen(c)
    wdwid, err := xproto.NewWindowId(c)
    if err != nil {
        return nil, err
    }

    lines := cutLines(c, gc.width - 2*gc.border, gc.font, title + " " + text)
    height := uint32(len(lines)) * gc.fontHeight

    var mask uint32 = xproto.CwBackPixel | xproto.CwOverrideRedirect | xproto.CwEventMask
    values := make([]uint32, 3)
    values[0] = scr.WhitePixel
    values[1] = 1
    values[2] = xproto.EventMaskExposure
    err = xproto.CreateWindowChecked(c, xproto.WindowClassCopyFromParent, wdwid, scr.Root,
                                     0, 0, uint16(gc.width), uint16(height + 2*gc.border), 1,
                                     xproto.WindowClassInputOutput, scr.RootVisual,
                                     mask, values).Check()
    if err != nil {
        return nil, err
    }
    xproto.ChangeProperty(c, xproto.PropModeReplace, wdwid,
                          xproto.AtomWmName, xproto.AtomString,
                          8, uint32(len(title)), []byte(title))
    xproto.MapWindow(c, wdwid)

    var wdw Window
    wdw.id     = wdwid
    wdw.conn   = c
    wdw.lines  = lines
    wdw.gc     = gc
    wdw.height = height
    return &wdw, nil
}

func (w *Window) Redraw() {
    /* Hide X11 borders */
    var mask uint16 = xproto.ConfigWindowBorderWidth
    values := make([]uint32, 1)
    values[0] = 0
    xproto.ConfigureWindow(w.conn, w.id, mask, values)

    wdt, hgh := int16(w.gc.width), int16(w.height + 2*w.gc.border)
    /* Drawing background */
    bg := xproto.Rectangle{0, 0, uint16(wdt), uint16(hgh)}
    bgs := make([]xproto.Rectangle, 1)
    bgs[0] = bg
    xproto.PolyFillRectangle(w.conn, xproto.Drawable(w.id), w.gc.bg, bgs)

    /* Drawing borders */
    vertices := make([]xproto.Point, 5)
    vertices[0].X = 0;   vertices[0].Y = 0
    vertices[1].X = wdt; vertices[1].Y = 0
    vertices[2].X = wdt; vertices[2].Y = hgh
    vertices[3].X = 0;   vertices[3].Y = hgh
    vertices[4].X = 0;   vertices[4].Y = 0
    xproto.PolyLine(w.conn, xproto.CoordModeOrigin, xproto.Drawable(w.id), w.gc.bc, vertices)

    /* Drawing text */
    hline := int16(w.gc.fontHeight)
    x, y := int16(w.gc.border), int16(w.gc.border + w.gc.fontUp)
    for _, line := range w.lines {
        xproto.ImageText8(w.conn, byte(len(line)), xproto.Drawable(w.id),
                          w.gc.fg, x, y, line)
        y += hline
    }
}

