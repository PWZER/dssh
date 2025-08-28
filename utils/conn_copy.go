package utils

import (
	"fmt"
	"io"
	"net"

	"github.com/PWZER/dssh/logger"
)

func CopyConn(src net.Conn, dstAddr string) error {
	defer src.Close()

	dst, err := net.Dial("tcp", dstAddr)
	if err != nil {
		return fmt.Errorf("Dial to %s failed: %v", dstAddr, err)
	}
	defer dst.Close()

	// dst -> src
	go func() {
		if _, err := io.Copy(dst, src); err != nil {
			logger.Errorf("Copy Error: %v", err)
		}
	}()

	// src -> dst
	if _, err := io.Copy(src, dst); err != nil {
		return fmt.Errorf("Copy Error: %v", err)
	}
	return nil
}
