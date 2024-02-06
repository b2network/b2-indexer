package rpc

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func HTTPPostJSON(proxyURL, httpURL, bodyJSON string) ([]byte, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		netTransport := &http.Transport{
			Proxy:                 http.ProxyURL(proxy),
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second * time.Duration(10),
		}
		httpClient.Transport = netTransport
	}
	b := strings.NewReader(bodyJSON)
	res, err := httpClient.Post(httpURL, "application/json", b)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
