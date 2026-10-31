package nbdinterpreter

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"encoding/binary"
	"github.com/makotonakai/nbd/types"
	"github.com/makotonakai/nbd/constants"
)

type NBDInterpreter struct {
    Cxn net.Conn
}

func NewNBDInterpreter(cxn net.Conn) (*NBDInterpreter, error) {
    ni := &NBDInterpreter{Cxn: cxn}

    // Send the initial handshake.
    if _, err := io.WriteString(cxn, "NBDMAGICIHAVEOPT"); err != nil {
        return nil, err
    }

    if _, err := cxn.Write([]byte{0x00, 0x01}); err != nil {
        return nil, err
    }

    // Read all four client flag bytes.
    buf := make([]byte, 4)
    if _, err := io.ReadFull(cxn, buf); err != nil {
        return nil, err
    }

    if !bytes.Equal(buf, constants.MinimalClientFlags) {
        return nil, fmt.Errorf("unknown client flags: %x", buf)
    }

    return ni, nil
}

func (ni *NBDInterpreter) GetClientOptions() ([]types.Option, error) {
    var options []types.Option

    for {
        prefix := make([]byte, 8)
        if _, err := io.ReadFull(ni.Cxn, prefix); err != nil {
            return nil, err
        }

        if !bytes.Equal(prefix, constants.OptionRequestPrefix) {
            return nil, fmt.Errorf("unknown option prefix: %x", prefix)
        }

        kind := make([]byte, 4)
        if _, err := io.ReadFull(ni.Cxn, kind); err != nil {
            return nil, err
        }

        lenBuf := make([]byte, 4)
        if _, err := io.ReadFull(ni.Cxn, lenBuf); err != nil {
            return nil, err
        }

        dataLen := binary.BigEndian.Uint32(lenBuf)

        // Prevent unbounded allocations from malformed input.
        if dataLen > 16*1024*1024 {
            return nil, fmt.Errorf("option payload too large: %d", dataLen)
        }

        data := make([]byte, int(dataLen))
        if _, err := io.ReadFull(ni.Cxn, data); err != nil {
            return nil, err
        }

        if bytes.Equal(kind, constants.OptionsExportName) {
            // The export-name option starts the transmission phase.
            // This prototype accepts the option but doesn't validate
            // or use the requested export name.
            return options, nil
        }

        option := types.Option{
            Kind: kind,
            Data: data,
        }

        // Fixed newstyle clients wait for a reply to each option
        // before sending the next one, so reply immediately.
        if err := ni.SendUnsupportedOption(option); err != nil {
            return nil, err
        }

        options = append(options, option)
    }
}

func (ni *NBDInterpreter) SendUnsupportedOption(option types.Option) error {
    response := make([]byte, 0, 20)

    response = append(response, constants.OptionResponsePrefix...)
    response = append(response, option.Kind...)
    response = append(response, constants.OptionUnsupported...)
    response = append(response, 0, 0, 0, 0)

    return writeFull(ni.Cxn, response)
}

func (ni *NBDInterpreter) SendExportResponse() error {
    response := make([]byte, 134)

    binary.BigEndian.PutUint64(response[0:8],
        uint64(constants.DefaultDeviceSize))

    // Advertise flush support so the client sends NBD_CMD_FLUSH.
    binary.BigEndian.PutUint16(response[8:10],
        constants.TransmissionFlagHasFlags|constants.TransmissionFlagSendFlush)

    // Bytes 10 through 133 remain zero.
    return writeFull(ni.Cxn, response)
}

func (ni *NBDInterpreter) GetTransmissionRequest() (*types.TransmissionRequest, error,) {
    prefix := make([]byte, 4)
    if _, err := io.ReadFull(ni.Cxn, prefix); err != nil {
        return nil, err
    }

    if !bytes.Equal(prefix, constants.RequestPrefix) {
        return nil, fmt.Errorf("unknown request prefix: %x", prefix)
    }

    header := make([]byte, 2+2+8+8+4)
    if _, err := io.ReadFull(ni.Cxn, header); err != nil {
        return nil, err
    }

    flags := header[0:2]
    if !bytes.Equal(flags, []byte{0, 0}) {
        return nil, fmt.Errorf("unexpected request flags: %x", flags)
    }

    kind := append([]byte(nil), header[2:4]...)
    handle := append([]byte(nil), header[4:12]...)
    offset := binary.BigEndian.Uint64(header[12:20])
    length := binary.BigEndian.Uint32(header[20:24])

    // Bound requests to the export's size.
    deviceSize := uint64(constants.DefaultDeviceSize)
    if offset > deviceSize || uint64(length) > deviceSize-offset {
        return nil, fmt.Errorf(
            "request out of bounds: offset=%d length=%d",
            offset, length,
        )
    }

    var data []byte

    if bytes.Equal(kind, constants.RequestKindWrite) && length > 0 {
        data = make([]byte, int(length))
        if _, err := io.ReadFull(ni.Cxn, data); err != nil {
            return nil, err
        }
    }

    return &types.TransmissionRequest{
        Kind:   kind,
        Handle: handle,
        Offset: offset,
        Length: length,
        Data:   data,
    }, nil
}

func (ni *NBDInterpreter) SendTransmissionResponse(handle, data []byte) error {
    response := make([]byte, 0, 16+len(data))
    response = append(response, constants.ResponsePrefix...)
    response = append(response, 0, 0, 0, 0) // success error code
    response = append(response, handle...)
    response = append(response, data...)

    return writeFull(ni.Cxn, response)
}

func writeFull(w io.Writer, data []byte) error {
    for len(data) > 0 {
        n, err := w.Write(data)
        if err != nil {
            return err
        }
        if n == 0 {
            return io.ErrShortWrite
        }
        data = data[n:]
    }
    return nil
}





