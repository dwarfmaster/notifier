package types

type Order interface {}

type KillOrder struct {}

type CloseOrder struct {
    All bool
    Top bool
    Id  uint32
}

type NotifOrder struct {
    Time  uint32
    Level string
    Text  string
}

type RedrawOrder struct {}

type Geometry struct {
    X, Y int32
    W, H int32
}

