package web

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const DefaultAddr = "127.0.0.1:19081"

func ResolveAddr(flagValue string) (string, error) {
	value := strings.TrimSpace(flagValue)
	if value == "" {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			if _, err := validPort(port); err != nil {
				return "", fmt.Errorf("PORT: %w", err)
			}
			return net.JoinHostPort("127.0.0.1", port), nil
		}
		value = DefaultAddr
	}
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return "", fmt.Errorf("监听地址必须是 host:port: %w", err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return "", fmt.Errorf("监听地址必须使用回环主机")
	}
	if _, err = validPort(port); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, port), nil
}

func validPort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1024 || port > 65535 {
		return 0, fmt.Errorf("端口必须是 1024 到 65535 的整数")
	}
	return port, nil
}
