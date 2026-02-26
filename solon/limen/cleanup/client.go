package cleanup

import (
	"errors"
	"fmt"
	"sync"
)

type ElasticCleaner interface {
	DeleteOrphan(username string) error
}

type VaultCleaner interface {
	DeleteOrphan(podName string) error
}

type Client struct {
	elastic ElasticCleaner
	vault   VaultCleaner
}

func NewClient(elastic ElasticCleaner, vault VaultCleaner) *Client {
	return &Client{
		elastic: elastic,
		vault:   vault,
	}
}

func (c *Client) DeleteOrphan(username, podName string) error {
	if c.elastic == nil {
		return fmt.Errorf("cleanup client is not initialized with an elastic cleaner")
	}
	if c.vault == nil {
		return fmt.Errorf("cleanup client is not initialized with a vault cleaner")
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.elastic.DeleteOrphan(username); err != nil {
			errCh <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.vault.DeleteOrphan(podName); err != nil {
			errCh <- err
		}
	}()

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
