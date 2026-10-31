package main

import (
    "bytes"
    "fmt"
    "io"
    "net"

    "github.com/makotonakai/nbd/constants"
    "github.com/makotonakai/nbd/nbdinterpreter"
)

func HandleConnection(cxn net.Conn, state []byte) {
    defer cxn.Close()

    ni, err := nbdinterpreter.NewNBDInterpreter(cxn)
    if err != nil {
        fmt.Println("Handshake failed:", err)
        return
    }

    options, err := ni.GetClientOptions()
    if err != nil {
        fmt.Println("Option negotiation failed:", err)
        return
    }

    for _, opt := range options {
        fmt.Printf("Ignored client option: %x\n", opt.Kind)
    }

    if err := ni.SendExportResponse(); err != nil {
        fmt.Println("Export response failed:", err)
        return
    }

    fmt.Println("Entering transmission phase")

    for {
        req, err := ni.GetTransmissionRequest()
        if err != nil {
            if err != io.EOF {
                fmt.Println("Request failed:", err)
            }
            return
        }

        start := int(req.Offset)
        end := start + int(req.Length)

        switch {
        case bytes.Equal(req.Kind, constants.RequestKindRead):
            fmt.Printf("Reading bytes %d - %d\n", start, end)

            if err := ni.SendTransmissionResponse(
                req.Handle, state[start:end],
            ); err != nil {
                fmt.Println("Read response failed:", err)
                return
            }

        case bytes.Equal(req.Kind, constants.RequestKindWrite):
            fmt.Printf("Writing bytes %d - %d\n", start, end)

            copy(state[start:end], req.Data)

            if err := ni.SendTransmissionResponse(
                req.Handle, nil,
            ); err != nil {
                fmt.Println("Write response failed:", err)
                return
            }

        case bytes.Equal(req.Kind, constants.RequestKindClose):
            return

        default:
            fmt.Printf("Unknown request type: %x\n", req.Kind)
            return
        }
    }
}

func main() {
    listener, err := net.Listen("tcp", ":2000")
    if err != nil {
        fmt.Println("Server start error:", err)
        return
    }
    defer listener.Close()

    // One fixed-size in-memory device.
    state := make([]byte, constants.DefaultDeviceSize)

    fmt.Println("NBD server starting...")

    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Connection error:", err)
            continue
        }

        go HandleConnection(conn, state)
    }
}
