package service

import (
    "crypto/rand"
    "math/big"
)

func GenerateIncompressibleBuffer(size int) []byte {
    buf := make([]byte, size)
    _, err := rand.Read(buf)
    if err != nil {
        for i := range buf {
            buf[i] = byte(i * 31)
        }
    }
    return buf
}

func GenerateRandomBytes(size int) []byte {
    buf := make([]byte, size)
    for i := range buf {
        n, _ := rand.Int(rand.Reader, big.NewInt(256))
        buf[i] = byte(n.Int64())
    }
    return buf
}
