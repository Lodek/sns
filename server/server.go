package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snsv1 "github.com/lodek/sns/gen/sns/v1"
	"github.com/lodek/sns/store"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type Server struct {
	snsv1.UnimplementedAlertServiceServer
	store *store.Store
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

func (s *Server) CreateOneShotAlert(ctx context.Context, req *snsv1.CreateOneShotAlertRequest) (*snsv1.CreateOneShotAlertResponse, error) {
	if req.FireAt == nil {
		return nil, status.Error(codes.InvalidArgument, "fire_at is required")
	}
	fireAt := req.FireAt.AsTime()
	if fireAt.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "fire_at must be in the future")
	}

	alert := &snsv1.OneShotAlert{
		Id:      uuid.NewString(),
		Name:    req.Name,
		Message: req.Message,
		FireAt:  timestamppb.New(fireAt),
	}
	if err := s.store.PutOneShotAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("store oneshot alert: %w", err)
	}
	return &snsv1.CreateOneShotAlertResponse{Alert: alert}, nil
}

func (s *Server) CreateRecurringAlert(ctx context.Context, req *snsv1.CreateRecurringAlertRequest) (*snsv1.CreateRecurringAlertResponse, error) {
	if req.CronExpression == "" {
		return nil, status.Error(codes.InvalidArgument, "cron_expression is required")
	}
	if _, err := cronParser.Parse(req.CronExpression); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cron expression: %v", err)
	}

	alert := &snsv1.RecurringAlert{
		Id:             uuid.NewString(),
		Name:           req.Name,
		Message:        req.Message,
		CronExpression: req.CronExpression,
	}
	if err := s.store.PutRecurringAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("store recurring alert: %w", err)
	}
	return &snsv1.CreateRecurringAlertResponse{Alert: alert}, nil
}

func (s *Server) ListAlerts(ctx context.Context, _ *snsv1.ListAlertsRequest) (*snsv1.ListAlertsResponse, error) {
	oneShots, err := s.store.ListOneShotAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list oneshot alerts: %w", err)
	}
	recurring, err := s.store.ListRecurringAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list recurring alerts: %w", err)
	}
	return &snsv1.ListAlertsResponse{
		OneShotAlerts:   oneShots,
		RecurringAlerts: recurring,
	}, nil
}

func (s *Server) DeleteAlert(ctx context.Context, req *snsv1.DeleteAlertRequest) (*snsv1.DeleteAlertResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.store.DeleteAlert(ctx, req.Id); err != nil {
		return nil, fmt.Errorf("delete alert: %w", err)
	}
	return &snsv1.DeleteAlertResponse{}, nil
}
