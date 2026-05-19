package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AuthCodeData — data yang disimpan di Redis saat OAuth authorize
type AuthCodeData struct {
	UserID      string `json:"user_id"`
	ClientID    string `json:"client_id"`
	Scope       string `json:"scope"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

type AuthCodeRepository interface {
	Save(code string, data *AuthCodeData, ttl time.Duration) error
	Find(code string) (*AuthCodeData, error)
	Delete(code string) error
}

type authCodeRepository struct {
	rdb *redis.Client
}

func NewAuthCodeRepository(rdb *redis.Client) AuthCodeRepository {
	return &authCodeRepository{rdb: rdb}
}

func (r *authCodeRepository) Save(code string, data *AuthCodeData, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("auth_code:%s", code)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return r.rdb.Set(ctx, key, jsonData, ttl).Err()
}

func (r *authCodeRepository) Find(code string) (*AuthCodeData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("auth_code:%s", code)

	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var data AuthCodeData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *authCodeRepository) Delete(code string) error {
	ctx := context.Background()
	key := fmt.Sprintf("auth_code:%s", code)
	return r.rdb.Del(ctx, key).Err()
}
