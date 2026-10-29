package audit

import (
	"bank-app-ioc/backend/internal/telemetry"
	"bank-app-ioc/backend/repository"
	"context"
	"github.com/google/uuid"
	"log"
	"net"
)

type Service struct{ repo *repository.AuditRepository }

func NewService(repo *repository.AuditRepository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, actor *uuid.UUID, eventType, action, result, ip string, correlation uuid.UUID, metadata map[string]string) {
	telemetry.CountAudit(ctx, eventType, result)
	if net.ParseIP(ip) == nil {
		ip = ""
	}
	if err := s.repo.Record(ctx, actor, eventType, action, result, ip, correlation, metadata); err != nil {
		log.Printf("audit event write failed event_type=%s correlation_id=%s: %v", eventType, correlation, err)
	}
}
