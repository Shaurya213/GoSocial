package common

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var publicMethods = map[string]bool {
	"/api.v1.UserService/Register": true,
	"/api.v1.UserService/Login" : true,
}


//grpc.UnaryServerInterceptor is middleware, for inspect incoming requests, enforce auth, log activity, modify context and then finally returns context and req to handler
//it returns a function signature, expected by gRPC for all request responses

//ctx.context for metadatanand request scoped values
// req for rpc request payload(like json sent by client)
//info contains methods name being called
// handler rpc function to call after auth
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error){
		
		// Bypass Auth for Public Methods
		//Full method is string - like "/api.v1.UserService/Register"
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		//extract auth header
		//extracting metadata from incoming context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		// taking authorizarization metadata having token
		vals := md["authorization"]
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization required!!")
		}

		// vals[0] = Bearer <token>
		// len(parts) = 2 
		// parse and validate format
		parts := strings.Fields(vals[0])
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
   				 return nil, status.Error(codes.Unauthenticated, "invalid auth header")
		}
		tokenString := parts[1]


		//validating jwt
		Claims, err := ValidToken(tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token!!")
		}


		//inject user identity into context
		ctx = context.WithValue(ctx, "user_id", Claims.UserID)
		ctx = context.WithValue(ctx, "handle", Claims.Handle)

		return handler(ctx, req)
	}
}


// so what i did in middleware-
// check if the methid is public 
// read authorization from metadata
// extract token from Bearer <token>
// validating token
// injecting userID/handle in context
// pass context and call handler