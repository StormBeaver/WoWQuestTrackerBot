package myproxy

import (
	"net/http"

	"golang.org/x/net/proxy"
)

func MakeProxyClient(name, pass, address string) (*http.Client, error) {
	auth := &proxy.Auth{User: name, Password: pass}

	dialer, err := proxy.SOCKS5("tcp", address, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
	}
	return client, nil
}
