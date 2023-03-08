package gocloudid

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func httpGet(url string) (res *http.Response, err error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return res, err
	}

	res, err = client.Do(req)
	if err != nil {
		return res, err
	}

	return res, nil
}

func get(url string) (body []byte, err error) {
	res, err := httpGet(url)
	if err != nil {
		return body, err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return body, fmt.Errorf(fmt.Sprintf("got api %s error, error code: %d.", url, res.StatusCode))
	}

	return io.ReadAll(res.Body)
}
