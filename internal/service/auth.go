package service

import (
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/mainak58/go_backend/internal/server"
)

type AuthService struct {
	server *server.Server
}

func NewAuthServices(s *server.Server) *AuthService {
	clerk.SetKey(s.Config.Auth.SecretKey)

	return &AuthService{
		server: s,
	}
}
