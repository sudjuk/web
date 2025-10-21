package handler

import (
    "front_start/internal/app/repository"
    "front_start/internal/app/redis"
)

type Handler struct {
	Repository  *repository.Repository
	RedisClient *redis.Client
	JWTSecret   string
}

func NewHandler(r *repository.Repository, redisClient *redis.Client, jwtSecret string) *Handler {
	return &Handler{
		Repository:  r,
		RedisClient: redisClient,
		JWTSecret:   jwtSecret,
	}
}
