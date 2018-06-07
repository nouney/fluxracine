package redis

import (
	"github.com/go-redis/redis"
	"github.com/nouney/fluxracine/internal/db"
)

// Redis is a redis client.
type Redis struct {
	client *redis.Client
}

// New creates a new Redis client.
func New(addr, password string) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	return &Redis{client: client}, nil
}

// AssignServer assigns a server to a user
func (r Redis) AssignServer(nickname, addr string) error {
	err := r.client.Set(nickname, addr, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// GetServer retrieves the server associated to the user
func (r Redis) GetServer(nickname string) (string, error) {
	addr, err := r.client.Get(nickname).Result()
	if err != nil {
		if err == redis.Nil {
			return "", db.ErrNotFound
		}
		return "", err
	}
	return addr, nil
}

// UnassignServer un-assigns a server from a user
func (r Redis) UnassignServer(nickname string) error {
	err := r.client.Del(nickname).Err()
	return err
}
