package redis

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Spiritreader/avior-go/cache"
	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
	"github.com/redis/go-redis/v9"
)

var RedisNotInitialized error = errors.New("redis client not initialized")
var RedisNoSubscription error = errors.New("no subscription found")
var instance *Redis
var once sync.Once

type Redis struct {
	Handle *Handle
}

type Handle struct {
	client   *redis.Client
	pubSub   *redis.PubSub
	cfg      *config.Data
	cancel   context.CancelFunc
	context  context.Context
	loopChan chan string
	running  atomic.Bool
}

func Get() *Redis {
	once.Do(func() {
		instance = new(Redis)
		instance.configure()
	})
	return instance
}

// Connect establishes a connection to the redis server
func (i *Redis) configure() {
	if i.Handle == nil || !i.Handle.running.Load() {
		cfg := config.Instance()
		ctx, cancel := context.WithCancel(context.Background())
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Local.Redis.Host,
			Password: cfg.Local.Redis.Password,
			DB:       cfg.Local.Redis.DB,
		})
		i.Handle = &Handle{client: rdb, cfg: cfg, cancel: cancel, context: ctx, loopChan: make(chan string, 1)}
	}
}

// PushMessage pushes a message to the redis broadcast job channel
func (r *Handle) PushMessage(msg string) error {
	if r.client == nil {
		return RedisNotInitialized
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return r.client.Publish(ctx, config.Instance().Local.Redis.ChannelPrefix+"-jobs", msg).Err()
}

// Closes the redis connection and stops pubsub and releases all resources
func (r *Handle) Close() {
	if !r.running.Load() {
		return
	}

	if r.pubSub != nil {
		err := r.pubSub.Close()
		if err != nil {
			glg.Warnf("redis: error closing pubsub while closing connection: %s", err)
		}
	}
	if r.client != nil {
		err := r.client.Close()
		if err != nil {
			glg.Errorf("redis: error during disconnect: %s", err)
		}
	}
	r.cancel()

	select {
	case <-r.loopChan:
	case <-time.After(2 * time.Second):
		glg.Warnf("redis: job broadcast subscription did not stop in time")
	}
}

// Running returns true if the redis connection is established and the pubsub is running
func (r *Handle) Running() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := r.client.Ping(ctx).Err()
	if err != nil {
		glg.Debugf("redis: client not responding, %s", err)
		return false
	}
	return r.running.Load()
}

// Subscribes to the redis job broadcast channel
func (r *Handle) subscribe() {
	if !r.running.Load() {
		r.pubSub = r.client.Subscribe(r.context, r.cfg.Local.Redis.ChannelPrefix+"-jobs")
		glg.Infof("redis: starting broadcast subscription to %s", r.cfg.Local.Redis.ChannelPrefix+"-jobs")
		go r.receiveLoop()
	}
}

func (r *Handle) receiveLoop() {
	r.running.Store(true)
	cache := cache.Instance()
	for {
		if r.context.Err() != nil {
			glg.Infof("redis: stopping job caching")
			r.running.Store(false)
			r.loopChan <- "stopped"
			return
		}
		ch := r.pubSub.Channel()
		for msg := range ch {
			glg.Infof("redis: received job %s", msg.Payload)
			// doesn't set LastUpdated because this is a broadcast
			// meaning there is no full refresh of the cache and data is inserted from non fs operations
			cache.Library.Data = append(cache.Library.Data, msg.Payload)
		}
	}
}

// AutoManage manages the redis connection based on the config
// Initializes and deinitializes pubsub
func AutoManage(prevCfg config.Redis) error {
	cfg := config.Instance()
	redis := Get()

	if cfg.Local.Redis.Enabled {
		if prevCfg.Enabled && (prevCfg.Host != cfg.Local.Redis.Host ||
			prevCfg.Password != cfg.Local.Redis.Password ||
			prevCfg.DB != cfg.Local.Redis.DB ||
			prevCfg.ChannelPrefix != cfg.Local.Redis.ChannelPrefix) {
			// config changed, close connection and reinitialize
			glg.Infof("redis: config changed, reconfiguring client")
			redis.Handle.Close()
		}
		redis.configure()
		redis.Handle.subscribe()
	} else {
		redis.Handle.Close()
	}
	return nil
}
