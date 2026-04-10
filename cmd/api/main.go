package main

import (
	"data_agent/internal/config"
	dataBase "data_agent/internal/db"
	"data_agent/internal/grpcserver"
	"data_agent/proto"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// main function to start the gRPC server
func main() {
	// initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// initialize database
	db, err := dataBase.InitDB()
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// load configuration
	cfg := config.LoadConfig()
	grpcPort := cfg.GRPCPort

	// start gRPC server
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		slog.Error("Failed to listen", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterHostServiceServer(grpcServer, &grpcserver.HostService{DB: db})
	proto.RegisterMetricServiceServer(grpcServer, &grpcserver.MetricService{DB: db})
	reflection.Register(grpcServer)

	go func() {
		slog.Info("gRPC server started", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("Failed to serve gRPC server", "error", err)
			os.Exit(1)
		}
	}()

	// handle termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// wait for termination signal
	<-stop
	slog.Info("Stopping gRPC server...")
	grpcServer.GracefulStop()
}
