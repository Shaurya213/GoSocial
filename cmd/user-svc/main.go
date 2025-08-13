package main

import (
	"fmt"
	"gosocial/internal/common"
	"gosocial/internal/dbmysql"
	"gosocial/internal/di"
	"log"
	"net"

	pb "gosocial/api/v1/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	fmt.Println("Initialiszing user service...")
	app, err := di.InitializeUserHandler()
	if err != nil {
		log.Fatalf("Failed to initialize chat service: %v", err)
	}

	if err := app.DB.AutoMigrate(&dbmysql.Device{},
	&dbmysql.User{}, &dbmysql.Friend{}); err != nil {
		log.Fatalf("Nahi chal raha AutoMigrate: %v", err)
	}
	log.Println("✅ Database migration completed")

	server := grpc.NewServer(
		//jwt authentcation middleware
		// it protects all the endpoints , except register/login
		//auth interceptor tera middleware hai
		grpc.UnaryInterceptor(common.AuthInterceptor()),
	)

	// step-3
	//initializing all the dependencies using wire, we are
	// using woire bcz wire figure out all the dependencies by itself

	//step-4
	//creating gRPC server with middleware
	// server ko middleware ke sath creating, bcz midlleware token parsing aur only alloowed servies ko bhi pass hone dega

	//step-5
	//regiser UserService with grpc Server, this makes handlers method available to gRPC requests
	pb.RegisterUserServiceServer(server, app.Handler)
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

	//step - 8
	listener, err := net.Listen("tcp", ":"+app.Config.Server.UserServicePort)
	log.Println("server bhai listening")
	if err != nil {
		log.Fatalf("Failegd to listen on port %s: %v", app.Config.Server.UserServicePort, err)
	}
	//allow server to run
	log.Println("User Service is Running bhai")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}

}


