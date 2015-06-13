package order

type Order interface {}

type KillOrder struct {}

type CloseOrder struct {
    All bool
}

type NotifOrder struct {
    time  uint32
    level string
    text  string
}

