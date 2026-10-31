package types

type Option struct {
    Kind []byte
    Data []byte
}

type TransmissionRequest struct {
    Kind   []byte
    Handle []byte
    Offset uint64
    Length uint32
    Data   []byte
}
