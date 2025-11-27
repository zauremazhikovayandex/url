package grpcserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zauremazhikovayandex/url/internal/auth"
	"github.com/zauremazhikovayandex/url/internal/grpc/pb"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"github.com/zauremazhikovayandex/url/internal/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedShortenerServer
	svc services.Shortener
}

func New(svc services.Shortener) *Server {
	return &Server{svc: svc}
}

func userIDFromContext(ctx context.Context) string {
	if uid := auth.GetUserID(ctx); uid != "" {
		return uid
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-user-id"); len(v) > 0 && v[0] != "" {
			return v[0]
		}
	}
	return "grpc-anonymous"
}

// ---- RPC methods ----

func (s *Server) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	if req == nil || strings.TrimSpace(req.Url) == "" {
		return nil, status.Error(codes.InvalidArgument, "empty url")
	}

	userID := userIDFromContext(ctx)
	shortURL, conflict, err := s.svc.CreateShortURL(ctx, req.Url, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidURL):
			return nil, status.Error(codes.InvalidArgument, "invalid url")
		default:
			return nil, status.Error(codes.Internal, "storage error")
		}
	}

	return &pb.ShortenResponse{
		ShortUrl: shortURL,
		Conflict: conflict,
	}, nil
}

func (s *Server) ShortenBatch(ctx context.Context, req *pb.BatchShortenRequest) (*pb.BatchShortenResponse, error) {
	userID := userIDFromContext(ctx)

	items := make([]services.BatchItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, services.BatchItem{
			CorrelationID: it.CorrelationId,
			OriginalURL:   strings.TrimSpace(it.OriginalUrl),
		})
	}

	results, err := s.svc.CreateShortURLBatch(ctx, items, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}

	out := make([]*pb.BatchResult, 0, len(results))
	for _, res := range results {
		out = append(out, &pb.BatchResult{
			CorrelationId: res.CorrelationID,
			ShortUrl:      res.ShortURL,
		})
	}
	return &pb.BatchShortenResponse{Results: out}, nil
}

func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if req == nil || strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing id")
	}

	orig, deleted, err := s.svc.Resolve(ctx, req.Id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "url not found")
		}
		return nil, status.Error(codes.Internal, "server error")
	}
	return &pb.GetResponse{OriginalUrl: orig, Deleted: deleted}, nil
}

func (s *Server) UserURLs(ctx context.Context, _ *pb.UserURLsRequest) (*pb.UserURLsResponse, error) {
	userID := userIDFromContext(ctx)
	urls, err := s.svc.UserURLs(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}
	out := make([]*pb.URLPair, 0, len(urls))
	for _, u := range urls {
		out = append(out, &pb.URLPair{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		})
	}
	return &pb.UserURLsResponse{Urls: out}, nil
}

func (s *Server) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	userID := userIDFromContext(ctx)
	if len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty ids")
	}
	if err := s.svc.DeleteUserURLs(ctx, req.Ids, userID); err != nil {
		return nil, status.Error(codes.Internal, "delete failed")
	}
	return &pb.DeleteUserURLsResponse{}, nil
}

func (s *Server) Ping(ctx context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	if err := s.svc.PingDB(ctx); err != nil {
		return nil, status.Error(codes.Unavailable, "db unavailable")
	}
	return &pb.PingResponse{}, nil
}

func (s *Server) Stats(ctx context.Context, _ *pb.StatsRequest) (*pb.StatsResponse, error) {
	var ipStr string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("x-real-ip")
		if len(vals) > 0 {
			ipStr = strings.TrimSpace(vals[0])
		}
	}
	ip := net.ParseIP(ipStr)

	urls, users, err := s.svc.Stats(ctx, ip)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
		return nil, status.Error(codes.Internal, "server error")
	}
	return &pb.StatsResponse{Urls: int32(urls), Users: int32(users)}, nil
}

func unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	if err != nil {
		logger.Log.Error(&message.LogMessage{Message: fmt.Sprintf("[gRPC] %s ERR: %v", info.FullMethod, err)})
	} else {
		logger.Log.Info(&message.LogMessage{Message: fmt.Sprintf("[gRPC] %s OK in %s", info.FullMethod, time.Since(start))})
	}
	return resp, err
}

// Launcher
func Serve(svc services.Shortener, ln net.Listener, useTLS bool, tlsCfg *tls.Config) error {
	var s *grpc.Server
	if useTLS {
		creds := credentials.NewTLS(tlsCfg)
		s = grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(unaryInterceptor))
	} else {
		s = grpc.NewServer(grpc.Creds(insecure.NewCredentials()), grpc.UnaryInterceptor(unaryInterceptor))
	}
	pb.RegisterShortenerServer(s, New(svc))
	return s.Serve(ln)
}
