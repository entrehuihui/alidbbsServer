package myrpc

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"mydnns/service/myrpc/middleware"
	"mydnns/service/myrpc/proto"
	"mydnns/service/myrpc/service"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type config struct {
	CertPemPath string
	CertKeyPath string
	grpcPort    string
	serverName  string
}

func loadConfig() config {
	return config{
		CertPemPath: viper.GetString("Certs.serverDir") + "tls.pem",
		CertKeyPath: viper.GetString("Certs.serverDir") + "tls.key",
		grpcPort:    ":" + strconv.Itoa(viper.GetInt("Server.port")),
		serverName:  viper.GetString("Server.serverName"),
	}
}

// GetTLSConfig .
func GetTLSConfig(certPemPath, certKeyPath string) *tls.Config {
	var certKeyPair *tls.Certificate
	cert, err := os.ReadFile(certPemPath)
	if err != nil {
		log.Fatal(err)
	}
	key, err := os.ReadFile(certKeyPath)
	if err != nil {
		log.Fatal(err)
	}

	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.Println("TLS KeyPair err: ", err)
	}
	certKeyPair = &pair

	return &tls.Config{
		Certificates: []tls.Certificate{*certKeyPair},
		NextProtos:   []string{http2.NextProtoTLS},
		ServerName:   viper.GetString("Server.serverName"),
	}
}

// StartServer 启动GRPC相关服务
func StartServer() {
	cfg := loadConfig()

	tlsConfig := GetTLSConfig(cfg.CertPemPath, cfg.CertKeyPath)
	grpcServer := startGRPC(cfg, tlsConfig)
	gwServer := startGW(cfg)

	conn, err := net.Listen("tcp", cfg.grpcPort)
	if err != nil {
		log.Fatal(err)
	}
	srv := http.Server{
		Addr:      cfg.grpcPort,
		Handler:   GrpcHandlerFunc(grpcServer, gwServer),
		TLSConfig: tlsConfig,
	}
	log.Println("服务启动")
	srv.Serve(tls.NewListener(conn, tlsConfig))
}

// GrpcHandlerFunc ..
func GrpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			allowCORS(otherHandler).ServeHTTP(w, r)
		}
	})
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept", "Authorization"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
}
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func startGRPC(cfg config, tlsConfig *tls.Config) *grpc.Server {
	creds, err := credentials.NewServerTLSFromFile(cfg.CertPemPath, cfg.CertKeyPath)
	if err != nil {
		panic(err)
	}
	// 中间件
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		// 新版
		grpc.ChainUnaryInterceptor(
			otelgrpc.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(middleware.InterceptorLogger()),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(middleware.CustomFunc)),
		),
		grpc.ChainStreamInterceptor(
			otelgrpc.StreamServerInterceptor(),
			logging.StreamServerInterceptor(middleware.InterceptorLogger()),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(middleware.CustomFunc)),
		),
	}
	server := service.NewService()
	gs := grpc.NewServer(opts...)

	// 注入GRPC服务
	proto.RegisterDnnsServer(gs, server)
	// 注入GRPC服务

	log.Println("准备启动GRPC服务", cfg.grpcPort)
	return gs
}
func startGW(cfg config) *http.ServeMux {
	ctx := context.Background()
	dcreds, err := credentials.NewClientTLSFromFile(cfg.CertPemPath, cfg.serverName)
	if err != nil {
		log.Fatal(err)
	}

	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(dcreds)}

	// 注入GW服务
	err = proto.RegisterDnnsHandlerFromEndpoint(ctx, mux, cfg.grpcPort, opts)
	if err != nil {
		log.Fatal("启动GW错误:", err)
	}
	// 注入GW服务

	gwmux := http.NewServeMux()
	gwmux.Handle("/v1/", mux)
	gwmux.Handle("/", http.FileServer(http.Dir("./dist")))
	gwmux.Handle("/static", http.FileServer(http.Dir("./static")))
	serveSwagger(gwmux)

	return gwmux
}

// swagger UI
func serveSwagger(mux *http.ServeMux) {
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(proto.AiasJSON))
	})
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir("./swagger"))))
}
