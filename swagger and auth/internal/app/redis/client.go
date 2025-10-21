package redis

import (
    "context"
    "fmt"
    "strconv"
    "time"

    appcfg "front_start/internal/app/config"
    "github.com/go-redis/redis/v8"
)

const sessionPrefix = "session:"

type Client struct {
    client *redis.Client
}

func New(ctx context.Context, cfg appcfg.Config) (*Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         cfg.Redis.Host + ":" + strconv.Itoa(cfg.Redis.Port),
        Password:     cfg.Redis.Password,
        DB:           0,
        DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Second,
        ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Second,
    })

    // Проверяем подключение
    if _, err := client.Ping(ctx).Result(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    return &Client{client: client}, nil
}

func (c *Client) SaveSession(ctx context.Context, sessionID string, userID int64, ttl time.Duration) error {
    key := sessionPrefix + sessionID
    return c.client.Set(ctx, key, userID, ttl).Err()
}

func (c *Client) GetUserIDBySession(ctx context.Context, sessionID string) (int64, error) {
    key := sessionPrefix + sessionID
    result := c.client.Get(ctx, key)
    if result.Err() != nil {
        return 0, result.Err()
    }
    
    userIDStr, err := result.Result()
    if err != nil {
        return 0, err
    }
    
    userID, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid user ID in session: %w", err)
    }
    
    return userID, nil
}

func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
    key := sessionPrefix + sessionID
    return c.client.Del(ctx, key).Err()
}

func (c *Client) Close() error {
    return c.client.Close()
}
