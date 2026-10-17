package audit

import (
	"context"
	"log"
	"net"

	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

type Service struct{ repo *repository.AuditRepository }

func NewService(repo *repository.AuditRepository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, actor *uuid.UUID, eventType, action, result, ip string, correlation uuid.UUID, metadata map[string]string) {
	if net.ParseIP(ip) == nil {
		ip = ""
	}
	if err := s.repo.Record(ctx, actor, eventType, action, result, ip, correlation, metadata); err != nil {
		log.Printf("audit event write failed event_type=%s correlation_id=%s: %v", eventType, correlation, err)
	}
}
