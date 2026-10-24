package store

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository/mock"
)

func TestRequestContextsAreIsolatedAndCanceled(t *testing.T) {
	type requestKey struct{}
	repo := &mock.SessionRepo{GetFn: func(ctx context.Context, id string) (*model.Session, error) {
		if ctx.Value(requestKey{}) != id {
			t.Errorf("context crossed requests: %s", id)
		}
		if ctx.Err() != context.Canceled {
			t.Error("cancellation lost")
		}
		return nil, nil
	}}
	shared := NewDynamoDBStore(repo)
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			id := fmt.Sprint(i)
			ctx, cancel := context.WithCancel(context.WithValue(context.Background(), requestKey{}, id))
			cancel()
			_, err := shared.WithContext(ctx).FetchValidUserSession(id)
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	group.Wait()
	if shared.ctx.Err() != nil || shared.ctx.Value(requestKey{}) != nil {
		t.Fatal("shared adapter mutated")
	}
}
