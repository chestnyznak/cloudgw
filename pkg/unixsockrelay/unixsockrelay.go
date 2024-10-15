package unixsockrelay

import (
	"fmt"
	"io"
	"net"
	"os"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

// UnixToTCPRelay makes unix to/from TCP socket relay
func UnixToTCPRelay(unixSockFile, hostAddr, tcpPort string) {
	defer func() {
		_ = os.Remove(unixSockFile)
	}()

	lsn, err := net.Listen("unix", unixSockFile)
	if err != nil {
		logger.Fatal("failed to start listen unix socket", "unix socket", unixSockFile, "error", err)
	}

	logger.Info("start listening unix socket", "unix socket", unixSockFile)

	for {
		unixConn, err := lsn.Accept()
		if err != nil {
			logger.Error("failed to accept data from unix socket", "error", err)

			continue
		}

		logger.Debug("new unix socket connection created")

		go unixToTCPDataForwarder(unixConn, hostAddr, tcpPort)
	}
}

// unixToTCPDataForwarder forwards unix sock to/from TCP sock
func unixToTCPDataForwarder(unixConn net.Conn, hostAddr, tcpPort string) {
	defer func() {
		if err := unixConn.Close(); err != nil {
			logger.Error("failed to close unix socket", "error", err)
		}

		logger.Info("disconnected from unix socket")
	}()

	tcpConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", hostAddr, tcpPort))
	if err != nil {
		logger.Error("failed to connect to tcp socket", "error", err)

		return
	}

	logger.Debug("connected to remote address", "remote address", tcpConn.RemoteAddr())

	go func() {
		if _, err := io.Copy(unixConn, tcpConn); err != nil {
			logger.Error("failed to copy from unix sock to tcp socket", "error", err)
		}
	}()

	if _, err = io.Copy(tcpConn, unixConn); err != nil {
		logger.Error("failed to copy from tcp socket to unix socket", "errior", err)
	}
}
