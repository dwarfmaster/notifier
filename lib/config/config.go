package config

import (
    "fmt"
    "os"
    "flag"
    "strings"
    "strconv"
    "bufio"
    "io"
)

func ConfigPath() (path string) {
    flag.StringVar(&path, "config", "", "path to the config file")
    flag.Parse()
    if path != "" {
        return
    }

    dir := os.Getenv("XDG_CONFIG_HOME")
    stat, err := os.Stat(dir)
    if err == nil && stat.IsDir() {
        path = dir + "/xcbnotif/config"
        stat, err = os.Stat(path)
        if err == nil && !stat.IsDir() {
            return
        }
    }

    return os.Getenv("HOME") + "/.xcbnotif"
}

type config struct {
    name string
    value string
    child *config
    next *config
}
var root *config

func parseKey(key string) []string {
    return strings.Split(key, ".")
}

func findOnLevel(name string, lvl *config) *config {
    entry := lvl
    for entry != nil {
        if entry.name == name {
            return entry
        }
        entry = entry.next
    }
    return nil
}

func followTree(path []string, rt *config, create bool) *config {
    if len(path) == 0 {
        return rt
    }

    entry := findOnLevel(path[0], rt.child)
    if entry != nil {
        return followTree(path[1:], entry, create)
    } else if !create {
        return nil
    } else {
        cfg := config{path[0], "", nil, nil}
        if rt.child == nil {
            rt.child = &cfg
        } else {
            cfg.next = rt.child
            rt.child = &cfg
        }
        return followTree(path[1:], rt.child, create)
    }
}

type InvalidLineError string
func (e InvalidLineError) Error() string {
    return fmt.Sprintf("Ill-formed config line : \"%s\"", string(e))
}

func parseLine(line string) error {
    var fields []string = strings.Fields(line)
    if len(fields) == 0 {
        return nil
    } else if len(fields) < 3 || fields[1] != ":" {
        return InvalidLineError(line)
    }

    var tree []string = parseKey(fields[0])
    cfg := followTree(tree, root, true)
    cfg.value = strings.Join(fields[2:], " ")
    return nil
}

func clearConfig() {
    root = nil
}

func Load(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    file.Seek(0, os.SEEK_SET)
    buffer := bufio.NewReader(file)
    root = &config{"root", "", nil, nil}

    for {
        line, err := buffer.ReadString('\n')
        switch err {
        case io.EOF:
            return nil
        case nil:
            err = parseLine(line)
            if err != nil {
                clearConfig()
                return err
            }
        default:
            clearConfig()
            return err
        }
    }
}

func dumpEntry(out io.Writer, lvl int, ent *config) {
    if ent == nil {
        return
    }
    fmt.Fprintf(out, "%v|-> %v : \"%v\"\n", strings.Repeat("\t", lvl), ent.name, ent.value)
    dumpEntry(out, lvl + 1, ent.child)
    dumpEntry(out, lvl, ent.next)
}

func Dump(out io.Writer) {
    dumpEntry(out, 0, root)
}

func Has(key string) bool {
    return followTree(parseKey(key), root, false) != nil
}

type NoEntryError string
func (e NoEntryError) Error() string {
    return fmt.Sprintf("No entry for \"%v\"", string(e))
}

type InvalidEntryError struct { key, value, err string }
func (e InvalidEntryError) Error() string {
    return fmt.Sprintf("Entry \"%v\" has invalid format : \"%v\" (%v)", e.key, e.value, e.err)
}

func String(key string) (string, error) {
    ent := followTree(parseKey(key), root, false)
    if ent == nil {
        return "", NoEntryError(key)
    } else {
        return ent.value, nil
    }
}

func Int(key string) (int32, error) {
    ent := followTree(parseKey(key), root, false)
    if ent == nil {
        return 0, NoEntryError(key)
    } else {
        i, e := strconv.ParseInt(ent.value, 0, 32)
        if e != nil {
            return 0, InvalidEntryError{key, ent.value, e.Error()}
        }
        return int32(i), nil
    }
}

func Bool(key string) (bool, error) {
    ent := followTree(parseKey(key), root, false)
    if ent == nil {
        return false, NoEntryError(key)
    } else {
        var b bool
        str := ent.value
        if str == "True" || str == "true" || str == "1" {
            b = true
        } else if str == "False" || str == "false" || str == "0" {
            b = false
        } else {
            return false, InvalidEntryError{key, str, "not a boolean"}
        }
        return b, nil
    }
}

