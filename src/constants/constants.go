package constants

const DefaultDeviceSize = 1 << 19 // 512 KiB

const DefaultDevicePath = "/export/blkdev"

const (
    TransmissionFlagHasFlags  = 1 << 0
    TransmissionFlagSendFlush = 1 << 2
)

var (
    MinimalClientFlags   = []byte{0x00, 0x00, 0x00, 0x01}
    OptionRequestPrefix  = []byte("IHAVEOPT")
    OptionResponsePrefix = []byte{0x00, 0x03, 0xe8, 0x89, 0x04, 0x55, 0x65, 0xa9}
    OptionsExportName    = []byte{0x00, 0x00, 0x00, 0x01}
    OptionUnsupported    = []byte{0x80, 0x00, 0x00, 0x01}
    RequestPrefix        = []byte{0x25, 0x60, 0x95, 0x13}
    RequestKindRead      = []byte{0x00, 0x00}
    RequestKindWrite     = []byte{0x00, 0x01}
    RequestKindClose     = []byte{0x00, 0x02}
    RequestKindFlush     = []byte{0x00, 0x03}
    ResponsePrefix       = []byte{0x67, 0x44, 0x66, 0x98}
)
