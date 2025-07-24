package user

import (
	"context"
	pb "gosocial/api/v1/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//it connects gPRC requests to our business logic(user_service.go)
// it helps to implement the services defined in the the .proto file and act as a layer between grpc request and service layer
// handler wires proto -> service
type Handler struct {
	pb.UnimplementedUserServiceServer
	userService UserService
}

func NewHandler(userService UserService) *Handler {
	return &Handler{userService: userService}
}

func(h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	user, token, err := h.userService.RegisterUser(ctx, req.Handle, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.AuthResponse{
		Token: token,
		UserId: int64(user.UserID),
		Message: "Registration successfull!!",
	}, nil
}


func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	user, token, err := h.userService.LoginUser(ctx, req.Handle, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid handle or Password")
	}
	return &pb.AuthResponse{
		Token: token,
		UserId: int64(user.UserID),
		Message: "Login Successfull",
	}, nil
}

func(h *Handler) GetProfile(ctx context.Context, req *pb.GetProfileRequest)(*pb.ProfileResponse, error) {
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

	user, err := h.userService.GetProfile(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.ProfileResponse{
		UserId: int64(user.UserID),
		Handle: user.Handle,
		Email: user.Email,
		Phone: user.Phone,
		ProfileDetails: user.ProfileDetails,
		Status: user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.StatusResponse, error) {
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	err := h.userService.UpdateProfile(ctx, userID, req.Email, req.Phone, req.ProfileDetails)
	if err != nil{
		return &pb.StatusResponse{Message: err.Error(), Success: false}, status.Error(codes.InvalidArgument,err.Error())
	}
	return &pb.StatusResponse{Message: "Profile Updated!", Success: true} , nil
}

func (h *Handler) SendFriendRequest(ctx context.Context, req *pb.FriendRequest) (*pb.StatusResponse, error) {
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	err := h.userService.SendFriendRequest(ctx, userID, uint64(req.TargetUserId))
	if err != nil {
		return &pb.StatusResponse{Message: err.Error(), Success: false}, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &pb.StatusResponse{Message: "Sent Friend Request", Success: true}, nil
}

func(h *Handler) AcceptFriendRequest(ctx context.Context, req *pb.FriendAcceptRequest) (*pb.StatusResponse, error) {
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	err := h.userService.AcceptFriendRequest(ctx, userID, uint64(req.RequesterId))
	if err != nil {
		return &pb.StatusResponse{Message: err.Error(), Success: false}, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &pb.StatusResponse{Message: "Friend Request Accepted!!", Success: true}, nil
}

func (h *Handler) ListFriends(ctx context.Context, req *pb.UserID) (*pb.FriendList, error){
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	friends, err := h.userService.ListFriends(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var out []*pb.Friend
	for _, u := range friends {
		out = append(out, &pb.Friend{
			UserId: int64(u.UserID),
			Handle: u.Handle,
			ProfileDetails: u.ProfileDetails,
			Status: u.Status,
		})
	}
	return &pb.FriendList{Friends: out}, nil

}


func (h *Handler) RegisterDevice(ctx context.Context, req *pb.DeviceTokenRequest) (*pb.StatusResponse, error) {
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	err := h.userService.RegisterDevice(ctx, userID, req.DeviceToken, req.Platform)
	if err != nil {
		return &pb.StatusResponse{Message: err.Error(), Success: false}, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.StatusResponse{Message: "Device Registered", Success: true}, nil
}

func(h *Handler) RemoveDevice(ctx context.Context, req *pb.DeviceTokenRequest) (*pb.StatusResponse, error) {
	_, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	err := h.userService.RemoveDevice(ctx, req.DeviceToken)
	if err != nil {
		return &pb.StatusResponse{Message: err.Error(), Success: false}, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.StatusResponse{Message: "Device Removed", Success: true}, nil
}

func(h *Handler) GetUserDevices(ctx context.Context, req *pb.UserID) (*pb.DeviceTokenList, error) {
	userID, ok := ctx.Value("user_id").(uint64)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }
	devices, err := h.userService.GetUserDevices(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var out []*pb.DeviceToken
	for _, d := range devices {
		out = append(out, &pb.DeviceToken{
			DeviceToken: d.DeviceToken,
			Platform: d.Platform,
		})
	}

	return &pb.DeviceTokenList{Devices: out}, nil
}


