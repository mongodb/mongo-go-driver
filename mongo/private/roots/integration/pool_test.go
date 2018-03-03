package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
)

func TestPool(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
		}
	}
	t.Run("Cannot Create Pool With Size Larger Than Capacity", func(t *testing.T) {
		_, err := connection.NewPool(addr.Addr(""), 4, 2)
		if err != connection.ErrSizeLargerThanCapacity {
			t.Errorf("Should not be able to create a pool with size larger than capacity. got %v; want %v", err, connection.ErrSizeLargerThanCapacity)
		}
	})
	t.Run("Reuses Connections", func(t *testing.T) {
		// TODO(skriptble): make this a table test.
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}

		c1, err := p.Get(context.Background())
		noerr(t, err)
		first := c1.ID()
		err = c1.Close()
		noerr(t, err)
		c2, err := p.Get(context.Background())
		noerr(t, err)
		second := c2.ID()
		if first != second {
			t.Errorf("Pool does not reuse connections. The connection ids differ. first %s; second %s", first, second)
		}
	})
	t.Run("Expired Connections Aren't Returned", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4,
			connection.WithIdleTimeout(func(time.Duration) time.Duration { return 300 * time.Millisecond }),
		)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		c1, err := p.Get(context.Background())
		noerr(t, err)
		first := c1.ID()
		err = c1.Close()
		noerr(t, err)
		time.Sleep(400 * time.Millisecond)
		c2, err := p.Get(context.Background())
		noerr(t, err)
		second := c2.ID()
		if first == second {
			t.Errorf("Pool does not reuse connections. The connection ids differe. first %s; second %s", first, second)
		}
	})
	t.Run("Get With Done Context", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = p.Get(ctx)
		if !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Expected context called error, but got: %v", err)
		}
	})
	t.Run("Get Returns Error From Creating A Connection", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr("localhost:0"), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		_, err = p.Get(context.Background())
		if !strings.Contains(err.Error(), "can't assign requested address") {
			t.Errorf("Expected context called error, but got: %v", err)
		}
	})
	t.Run("Get Returns An Error After Pool Is Closed", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Close()
		noerr(t, err)
		_, err = p.Get(context.Background())
		if err != connection.ErrPoolClosed {
			t.Errorf("Did not get expected error. got %v; want %v", err, connection.ErrPoolClosed)
		}
	})
	t.Run("Connection Close Does Not Error After Pool Is Closed", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		c1, err := p.Get(context.Background())
		noerr(t, err)
		err = p.Close()
		noerr(t, err)
		err = c1.Close()
		if err != nil {
			t.Errorf("Conneciton Close should not error after Pool is closed, but got error: %v", err)
		}
	})
	t.Run("Connection Close Does Not Close Underlying Connection If Not Expired", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Connection Close Does Close Underlying Connection If Expired", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Connection Close Closes Underlying Connection When Size Is Exceeded", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Drain Expires Existing Checked Out Connections", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		c1, err := p.Get(context.Background())
		noerr(t, err)
		if c1.Expired() != false {
			t.Errorf("Newly retrieved connection should not be expired.")
		}
		err = p.Drain()
		noerr(t, err)
		if c1.Expired() != true {
			t.Errorf("Existing checkout out connections should be expired once pool is drained.")
		}
	})
	t.Run("Drain Expires Idle Connections", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Close Is Idempotent", func(t *testing.T) {
		p, err := connection.NewPool(addr.Addr(*host), 2, 4)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Close()
		noerr(t, err)
		err = p.Close()
		if err != nil {
			t.Errorf("Should be able to call Close twice, but got error: %v", err)
		}
	})
	t.Run("Pool Close Closes All Connections In A Pool", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
}
