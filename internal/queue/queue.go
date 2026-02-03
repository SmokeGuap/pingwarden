package queue

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrEmpty = errors.New("queue empty")

const eventsKey = "pingwarden:events"

type Queue struct {
	rdb *redis.Client
	key string
}

func New(addr string) *Queue {
	return &Queue{
		rdb: redis.NewClient(&redis.Options{Addr: addr}),
		key: "pingwarden:jobs",
	}
}

func (q *Queue) Ping(ctx context.Context) error { return q.rdb.Ping(ctx).Err() }

func (q *Queue) Close() error { return q.rdb.Close() }

func (q *Queue) Push(ctx context.Context, ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}
	vals := make([]any, len(ids))
	for i, id := range ids {
		vals[i] = id
	}
	return q.rdb.LPush(ctx, q.key, vals...).Err()
}

func (q *Queue) Pop(ctx context.Context, timeout time.Duration) (int64, error) {
	res, err := q.rdb.BRPop(ctx, timeout, q.key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrEmpty
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(res[1], 10, 64)
}

func (q *Queue) PushEvent(ctx context.Context, payload []byte) error {
	return q.rdb.LPush(ctx, eventsKey, payload).Err()
}

func (q *Queue) PopEvent(ctx context.Context, timeout time.Duration) ([]byte, error) {
	res, err := q.rdb.BRPop(ctx, timeout, eventsKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrEmpty
	}
	if err != nil {
		return nil, err
	}
	return []byte(res[1]), nil
}
