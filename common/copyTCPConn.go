package common

import (
	"io"
	"net"
)

func CopyTCPConn(dst, src *net.TCPConn) {
	_, _ = io.Copy(dst, src)
	_ = src.CloseRead()
	_ = dst.CloseWrite()
}
