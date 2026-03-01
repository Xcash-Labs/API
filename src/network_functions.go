package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net"
	"time"
	"encoding/binary"
	"fmt"
)

func send_http_data(url string, data string) (string, error) {
	// create the http request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", errors.New("failed to create HTTP request")
	}

	// set the request headers
	req.Header.Set("Content-Type", "application/json")

	// set the http client settings
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	// send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("failed to send HTTP request")
	}
	defer resp.Body.Close()

	// get the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read response body")
	}

	return string(body), nil
}

func send_http_get(url string) (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sendTCPJSON(host string, port int, jsonReq string, timeout time.Duration) (string, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return "", fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	// 1) send request JSON (NO length prefix)
	if _, err := conn.Write([]byte(jsonReq)); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	// 2) read 4-byte big-endian response length
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return "", fmt.Errorf("read prefix failed: %w", err)
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 {
		return "", fmt.Errorf("bad response length (0)")
	}
	if n > 4*1024*1024 { // safety cap (4MB)
		return "", fmt.Errorf("response too large (%d bytes)", n)
	}

	// 3) read response body
	buf := make([]byte, n)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	return string(buf), nil
}