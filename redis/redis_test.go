package redis

import (
	"testing"
	"time"

	"github.com/Spiritreader/avior-go/cache"
	"github.com/Spiritreader/avior-go/config"
)

func TestRedis(t *testing.T) {
	config.LoadLocalFrom("../config.json")
	redis := Get()
	redis.configure()
	redis.Handle.subscribe()
	t.Logf("redis running: %t", redis.Handle.Running())

	msg := "test"
	err := redis.Handle.PushMessage(msg)
	if err != nil {
		t.Errorf("could not broadcast job: %s", err)
	}
	time.Sleep(time.Millisecond * 500)
	cache := cache.Instance()
	t.Logf("jobs %+v", cache.Library.Data)
	if len(cache.Library.Data) == 0 {
		t.Errorf("expected %s, got none", msg)
	} else if cache.Library.Data[0] != msg {
		t.Errorf("expected %s, got %s", msg, cache.Library.Data[0])
	}
	redis.Handle.Close()
}
