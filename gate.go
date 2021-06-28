package gate

import (
	"errors"
	"hash/crc32"
)

//

const (
	typeServer = 1
	typeClient = 2
)

const (
	OK = iota
	Err_TokenFailed
	Err_NoService
)

var errCode = map[uint32]error{
	Err_NoService:   errors.New("no service"),
	Err_TokenFailed: errors.New("token failed"),
}

func GetCode(code uint32) error {
	return errCode[code]
}

// type:1 id:4 tokenHash:4

func HashNumber(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}
