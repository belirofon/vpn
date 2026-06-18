package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func FetchSubscription(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}

		resp, err := client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("fetch %s: %w", url, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read %s: %w", url, err)
			continue
		}

		return body, nil
	}

	return nil, lastErr
}
