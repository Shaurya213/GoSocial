package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "gosocial/api/v1"
	"gosocial/internal/dbmysql"
	"gosocial/internal/di"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	log.Println("Initializing application...")
	app, err := di.InitializeApplication()

	if err := app.DB.AutoMigrate(&dbmysql.Notification{}, &dbmysql.Device{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("✅ Database migration completed")

	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterNotificationServiceServer(grpcServer, app.Handler.GRPC)

	reflection.Register(grpcServer) // used for services discovery , and enables grpcurl , postman services(without effecting proto files), and also helps to qurey the server for schema

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s",  app.Config.Server.NotifServicePort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// we are using goroutine here because we are running this function in a seprate goroutine from the main thread so when the server is stopped it
	//can easily be execute concurrently and continue executing apart from the main thread
	//It helps to prevent blocking  without this grpcServer.Serve(lis) would block the main thread indefinetly.

	go func() {
		log.Printf("gRPC server listening on %s", lis.Addr()) //here we are starting the server using go routine
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// communicates between goroutines , handles shutdown service , commincation between main thread and goroutine.

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) //we are using channels to shut down the server
	<-quit

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	if app.Service != nil {
		app.Service.Shutdown()
	}
	log.Println("Server stopped")
}
