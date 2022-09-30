package consts

import "errors"

var (
	ErrProtocal      = errors.New("protocal error")
	ErrProcessPacket = errors.New("process packet error")
)
