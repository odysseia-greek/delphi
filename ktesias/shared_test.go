package ktesias

import (
	"fmt"
	"time"
)

func retryWithTimeout(timeout, interval time.Duration, action func() error) error {
	deadline := time.Now().Add(timeout)
	for {
		err := action()
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s: %w", timeout, err)
		}

		time.Sleep(interval)
	}
}
