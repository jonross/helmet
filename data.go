package helmet

func ReadShort(buf []byte) (int) {
    bits := uint(buf[0]) <<  8 |
            uint(buf[1])
    return int(bits)
}

func GetInt32(buf []byte) (int32) {
    bits := uint32(buf[0]) << 24 |
            uint32(buf[1]) << 16 |
            uint32(buf[2]) <<  8 |
            uint32(buf[3])
    return int32(bits)
}

func GetUInt32(buf []byte) (uint32) {
    bits := uint32(buf[0]) << 24 |
            uint32(buf[1]) << 16 |
            uint32(buf[2]) <<  8 |
            uint32(buf[3])
    return bits
}

func GetUInt64(buf []byte) (uint64) {
    bits := uint64(buf[0]) << 54 |
            uint64(buf[1]) << 48 |
            uint64(buf[2]) << 40 |
            uint64(buf[3]) << 32 |
            uint64(buf[4]) << 24 |
            uint64(buf[5]) << 16 |
            uint64(buf[6]) <<  8 |
            uint64(buf[7])
    return bits
}

