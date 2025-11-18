package grpcserver

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/zauremazhikovayandex/url/internal/auth"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/postgres"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
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
	urlService services.URLService
}

func New(urlSvc services.URLService) *Server {
	return &Server{urlService: urlSvc}
}

// --- helpers ---

// isValidURL - Проверка на корректный URL
func isValidURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func userIDFromContext(ctx context.Context) string {
	// Пытаемся взять из HTTP-совместимого контекста
	if uid := auth.GetUserID(ctx); uid != "" {
		return uid
	}
	// Или из gRPC метаданных (authorization / x-user-id)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-user-id"); len(v) > 0 && v[0] != "" {
			return v[0]
		}
	}
	return "grpc-anonymous"
}

// generateShortID - Генерация ID
func generateShortID(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

// --- RPC methods ---

func (s *Server) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	if req == nil || strings.TrimSpace(req.Url) == "" {
		return nil, status.Error(codes.InvalidArgument, "empty url")
	}
	if !isValidURL(req.Url) {
		return nil, status.Error(codes.InvalidArgument, "invalid url")
	}

	userID := userIDFromContext(ctx)
	id, err := generateShortID(8)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate id")
	}

	st := config.AppConfig.StorageType
	if st == "DB" {
		if err := s.urlService.SaveURL(ctx, id, req.Url, userID); err != nil {
			if errors.Is(err, postgres.ErrDuplicateOriginalURL) {
				// найти существующий id
				existID, getErr := s.urlService.GetShortIDByOriginalURL(ctx, req.Url)
				if getErr != nil || existID == "" {
					return &pb.ShortenResponse{ShortUrl: "", Conflict: true}, nil
				}
				shortURL := fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, existID)
				return &pb.ShortenResponse{ShortUrl: shortURL, Conflict: true}, nil
			}
			return nil, status.Error(codes.Internal, "storage error")
		}
	} else {
		storage.Store.Set(id, req.Url)
	}

	return &pb.ShortenResponse{
		ShortUrl: fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id),
		Conflict: false,
	}, nil
}

func (s *Server) ShortenBatch(ctx context.Context, req *pb.BatchShortenRequest) (*pb.BatchShortenResponse, error) {
	userID := userIDFromContext(ctx)
	st := config.AppConfig.StorageType

	var out []*pb.BatchResult
	for _, it := range req.Items {
		url := strings.TrimSpace(it.OriginalUrl)
		if url == "" || !isValidURL(url) {
			continue
		}
		id, err := generateShortID(8)
		if err != nil {
			continue
		}
		if st == "DB" {
			if err := s.urlService.SaveURL(ctx, id, url, userID); err != nil {
				continue
			}
		} else {
			storage.Store.Set(id, url)
		}
		out = append(out, &pb.BatchResult{
			CorrelationId: it.CorrelationId,
			ShortUrl:      fmt.Sprintf("%s/%s", config.AppConfig.BaseURL, id),
		})
	}
	return &pb.BatchShortenResponse{Results: out}, nil
}

func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if req == nil || strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing id")
	}
	st := config.AppConfig.StorageType
	if st == "DB" {
		orig, err := s.urlService.GetOriginalURL(ctx, req.Id)
		if err != nil {
			if errors.Is(err, postgres.ErrURLDeleted) {
				return &pb.GetResponse{OriginalUrl: "", Deleted: true}, nil
			}
			return nil, status.Error(codes.NotFound, "url not found")
		}
		return &pb.GetResponse{OriginalUrl: orig, Deleted: false}, nil
	}
	// memory/file
	if orig, ok := storage.Store.Get(req.Id); ok && orig != "" {
		return &pb.GetResponse{OriginalUrl: orig, Deleted: false}, nil
	}
	return nil, status.Error(codes.NotFound, "url not found")
}

func (s *Server) UserURLs(ctx context.Context, _ *pb.UserURLsRequest) (*pb.UserURLsResponse, error) {
	userID := userIDFromContext(ctx)
	urls, err := s.urlService.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}
	if len(urls) == 0 {
		return &pb.UserURLsResponse{Urls: nil}, nil
	}
	out := make([]*pb.URLPair, 0, len(urls))
	for _, u := range urls {
		out = append(out, &pb.URLPair{
			ShortUrl:    config.AppConfig.BaseURL + "/" + u.ID,
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
	if err := s.urlService.BatchDelete(ctx, req.Ids, userID); err != nil {
		return nil, status.Error(codes.Internal, "delete failed")
	}
	return &pb.DeleteUserURLsResponse{}, nil
}

func (s *Server) Ping(ctx context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	conn, err := postgres.SQLInstance()
	if conn == nil || err != nil {
		return nil, status.Error(codes.Unavailable, "db unavailable")
	}
	return &pb.PingResponse{}, nil
}

func (s *Server) Stats(ctx context.Context, _ *pb.StatsRequest) (*pb.StatsResponse, error) {
	// Аналог /api/internal/stats: проверка trusted_subnet по X-Real-IP из metadata
	if config.AppConfig.TrustedIPNet == nil {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	var ipStr string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("x-real-ip")
		if len(vals) > 0 {
			ipStr = strings.TrimSpace(vals[0])
		}
	}
	ip := net.ParseIP(ipStr)
	if ipStr == "" || ip == nil || !config.AppConfig.TrustedIPNet.Contains(ip) {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	if config.AppConfig.StorageType == "DB" {
		u, us, err := postgres.CountStats(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, "server error")
		}
		return &pb.StatsResponse{Urls: int32(u), Users: int32(us)}, nil
	}

	if storage.Store == nil {
		return nil, status.Error(codes.Internal, "server error")
	}
	storage.Store.Mu.RLock()
	n := len(storage.Store.DataUnsafe())
	storage.Store.Mu.RUnlock()
	return &pb.StatsResponse{Urls: int32(n), Users: 0}, nil
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
func Serve(urlSvc services.URLService, ln net.Listener, useTLS bool, tlsCfg *tls.Config) error {
	var s *grpc.Server
	if useTLS {
		creds := credentials.NewTLS(tlsCfg)
		s = grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(unaryInterceptor))
	} else {
		s = grpc.NewServer(grpc.Creds(insecure.NewCredentials()), grpc.UnaryInterceptor(unaryInterceptor))
	}
	pb.RegisterShortenerServer(s, New(urlSvc))
	return s.Serve(ln)
}
