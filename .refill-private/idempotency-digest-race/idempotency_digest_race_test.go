package idempotency_digest_race_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"citytree/internal/application"
)

var errStopAfterDigest = errors.New("stop after digest")

type failingStore struct {
	application.Store
}

func (failingStore) GetIdempotency(context.Context, string, string) (application.IdempotencyRecord, error) {
	return application.IdempotencyRecord{}, errStopAfterDigest
}

func TestConcurrentCommandsSynchronizeDigestCache(t *testing.T) {
	service := application.NewService(failingStore{})
	const workers = 48
	const requestsPerWorker = 64

	start := make(chan struct{})
	ready := sync.WaitGroup{}
	ready.Add(workers)
	done := sync.WaitGroup{}
	done.Add(workers)

	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			for request := 0; request < requestsPerWorker; request++ {
				cmd := application.CreateBatchCommand{
					Name:     fmt.Sprintf("并发批次-%d-%d", worker, request),
					Area:     "并发测试区",
					Operator: "测试员",
				}
				_, err := service.CreateBatch(context.Background(), cmd, fmt.Sprintf("digest-race-%d-%d", worker, request))
				if !errors.Is(err, errStopAfterDigest) {
					t.Errorf("CreateBatch error = %v, want %v", err, errStopAfterDigest)
					return
				}
			}
		}()
	}

	ready.Wait()
	close(start)
	done.Wait()
}
