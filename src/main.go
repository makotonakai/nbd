package main

import (
    "bytes"
    "fmt"
    "io"
    "net"
    "os"

    "github.com/makotonakai/nbd/constants"
    "github.com/makotonakai/nbd/nbdinterpreter"
)

func HandleConnection(cxn net.Conn, device *os.File) {
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

            data := make([]byte, req.Length)
            if _, err := device.ReadAt(data, int64(req.Offset)); err != nil {
                fmt.Println("Read failed:", err)
                return
            }

            if err := ni.SendTransmissionResponse(
                req.Handle, data,
            ); err != nil {
                fmt.Println("Read response failed:", err)
                return
            }

        case bytes.Equal(req.Kind, constants.RequestKindWrite):
            fmt.Printf("Writing bytes %d - %d\n", start, end)

            if _, err := device.WriteAt(req.Data, int64(req.Offset)); err != nil {
                fmt.Println("Write failed:", err)
                return
            }

            if err := ni.SendTransmissionResponse(
                req.Handle, nil,
            ); err != nil {
                fmt.Println("Write response failed:", err)
                return
            }

        case bytes.Equal(req.Kind, constants.RequestKindFlush):
            fmt.Println("Flushing device")

            if err := device.Sync(); err != nil {
                fmt.Println("Flush failed:", err)
                return
            }

            if err := ni.SendTransmissionResponse(
                req.Handle, nil,
            ); err != nil {
                fmt.Println("Flush response failed:", err)
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

    // One fixed-size file-backed device.
    device, err := os.OpenFile(constants.DefaultDevicePath, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        fmt.Println("Device open error:", err)
        return
    }
    defer device.Close()

    // Grow or shrink the file to the advertised export size.
    if err := device.Truncate(constants.DefaultDeviceSize); err != nil {
        fmt.Println("Device resize error:", err)
        return
    }

    fmt.Println("NBD server starting...")

    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Connection error:", err)
            continue
        }

        go HandleConnection(conn, device)
    }
}
