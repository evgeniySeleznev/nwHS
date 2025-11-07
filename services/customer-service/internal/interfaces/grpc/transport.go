package grpc

import (
	"context"
	"net"
	"time"

	"github.com/company/holo/services/customer-service/internal/application/commands"
	appqueries "github.com/company/holo/services/customer-service/internal/application/queries"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Transport представляет gRPC-адаптер для customer-service.
type Transport struct {
	server          *grpc.Server
	registerHandler *commands.RegisterCustomerHandler
	getHandler      *appqueries.GetCustomerHandler
	log             *zap.Logger
}

// NewTransport создаёт gRPC сервер и навешивает middlewares (интерцепторы).
func NewTransport(register *commands.RegisterCustomerHandler, getter *appqueries.GetCustomerHandler, log *zap.Logger, opts ...grpc.ServerOption) *Transport {
	srv := grpc.NewServer(opts...)
	t := &Transport{
		server:          srv,
		registerHandler: register,
		getHandler:      getter,
		log:             log,
	}
	// TODO: при генерации protobuf зарегистрировать customerpb.RegisterCustomerServiceServer(srv, t)
	return t
}

// Serve запускает gRPC сервер.
func (t *Transport) Serve(lis net.Listener) error {
	t.log.Info("gRPC transport listening", zap.String("addr", lis.Addr().String()))
	return t.server.Serve(lis)
}

// Stop останавливает сервер плавно.
func (t *Transport) Stop() {
	t.server.GracefulStop()
}

// RegisterCustomer демонстрирует обработку RPC.
func (t *Transport) RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*RegisterCustomerResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	t.log.Debug("incoming metadata", zap.Any("metadata", md))

	id, err := t.registerHandler.Handle(ctx, commands.RegisterCustomer{
		FullName:    req.FullName,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		BirthDate:   req.BirthDate,
	})
	if err != nil {
		return nil, err
	}

	return &RegisterCustomerResponse{Id: id}, nil
}

// GetCustomer демонстрирует обработку query RPC.
func (t *Transport) GetCustomer(ctx context.Context, req *GetCustomerRequest) (*GetCustomerResponse, error) {
	dto, err := t.getHandler.Handle(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &GetCustomerResponse{
		Id:          dto.ID,
		FullName:    dto.FullName,
		Email:       dto.Email,
		PhoneNumber: dto.PhoneNumber,
	}, nil
}

// RegisterCustomerRequest описывает входящие данные RPC (заглушка до генерации protobuf).
type RegisterCustomerRequest struct {
	FullName    string
	Email       string
	PhoneNumber string
	BirthDate   time.Time
}

// RegisterCustomerResponse содержит идентификатор созданного клиента.
type RegisterCustomerResponse struct {
	Id string
}

// GetCustomerRequest содержит ID клиента.
type GetCustomerRequest struct {
	Id string
}

// GetCustomerResponse возвращает DTO клиента.
type GetCustomerResponse struct {
	Id          string
	FullName    string
	Email       string
	PhoneNumber string
}
