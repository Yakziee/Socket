package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func parseAddress(arg string) (string, error) {
	const defaultAddress = ":7777"

	arg = strings.TrimSpace(arg)
	if arg == "" {
		return defaultAddress, nil
	}

	if port, err := strconv.Atoi(arg); err == nil {
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("invalid port: %q", arg)
		}

		return ":" + arg, nil
	}

	_, port, err := net.SplitHostPort(arg)
	if err != nil {
		return "", fmt.Errorf("invalid address %q %w", arg, err)
	}

	n, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("invalid port %q %w", arg, err)
	}

	if n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid port %q", port)
	}

	return arg, nil
}
