package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"
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