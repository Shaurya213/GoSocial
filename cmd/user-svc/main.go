package main

import (
	"fmt"
	"gosocial/internal/common"
	"gosocial/internal/di"
	"log"
	"net"
	"os"

	pb "gosocial/api/v1/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("gosocial user main")
	
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system env variables")
	}
	
	//step-1
	//Load Configurations for database
	dbconfig := common.GetDatabaseConfig()

	//load configuration for server
	serverport := os.Getenv("SERVER_PORT")
	if serverport == "" {
		serverport = "8080"
	}

	log.Println("Configuration Loaded")

	//step-2
	//initializing database
	db, err := common.InitDatabase(dbconfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	log.Println("Database Initialized successfully")

	// step-3
	//initializing all the dependencies using wire, we are
	// using woire bcz wire figure out all the dependencies by itself
	handler := di.InitializeUserHandler(db)
	log.Println("Dependencies wired Successfully")

	//step-4
	//creating gRPC server with middleware
	// server ko middleware ke sath creating, bcz midlleware token parsing aur only alloowed servies ko bhi pass hone dega
	server := grpc.NewServer(
		//jwt authentcation middleware
		// it protects all the endpoints , except register/login
		//auth interceptor tera middleware hai
		grpc.UnaryInterceptor(common.AuthInterceptor()),
	)

	//step-5
	//regiser UserService with grpc Server, this makes handlers method available to gRPC requests
	pb.RegisterUserServiceServer(server, handler)
	log.Println("UserService Registered with gPRC server")

	//step - 6
	// swagger ka bhai hai ye
	// jab bhi client side puchta hai ki kya kya methods hai services mae, unke input types aur reponse type to ye
	//  bhai ki madad se krte hai, proto use nhi krna hota
	// kroge to tumhari wish hai, faltu kaam hard kar rahe ho
	reflection.Register(server)
	log.Println("gRPC reflection enabled")

	// step-7
	// ab jo server banaya hai, usse port pr listen krna hai
	// serverport pr saari rewuest aayegi
	// reflection saari input output of services ka format bta dega
	// server ke sath middleware hai- jo protected routes pr authentication ke baad corresponding handler
	// function of request pass krega
	listener, err := net.Listen("tcp", ":"+serverport)
	log.Println("server bhai listening")
	if err != nil {
		log.Fatalf("Failegd to listen on port %s: %v", serverport, err)
	}

	//step - 8
	//allow server to run
	log.Println("User Service is Running bhai")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}

}


