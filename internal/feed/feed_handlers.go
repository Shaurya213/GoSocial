package feed

import (
	"context"
	"time"

	feedpb "GoSocial/api/v1/feed" // alias the generated package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeedHandlers struct {
	FeedSvc FeedUsecase
	feedpb.UnimplementedFeedServiceServer
}

func (h *FeedHandlers) CreatePost(ctx context.Context, req *feedpb.CreatePostRequest) (*feedpb.FeedResponse, error) {
	if req.AuthorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid author ID")
	}
	if req.Text == "" && len(req.MediaData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "post must have text or file data")
	}
	if req.MediaType == "" {
		return nil, status.Error(codes.InvalidArgument, "media type must be specified")
	}
	if req.Privacy == "" {
		return nil, status.Error(codes.InvalidArgument, "privacy setting must be specified")
	}
	if req.MediaName == "" {
		return nil, status.Error(codes.InvalidArgument, "media name must be specified")
	}
	if len(req.MediaData) > 0 && req.MediaName == "" {
		return nil, status.Error(codes.InvalidArgument, "media name must be specified if media data is provided")
	}
	// Call the service method to create the post
	postID, err := h.FeedSvc.CreatePost(
		ctx,
		req.AuthorId,
		req.Text,
		req.MediaData,
		req.MediaName,
		req.MediaType,
		req.Privacy,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create post: %v", err)
	}

	return &feedpb.FeedResponse{
		ContentId: postID,
		Message:   "Post created successfully",
	}, nil
}

func (h *FeedHandlers) CreateReel(ctx context.Context, req *feedpb.CreateReelRequest) (*feedpb.FeedResponse, error) {
	if req.AuthorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid author ID")
	}
	if req.Caption == "" && len(req.MediaData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "reel must have caption or file data")
	}
	if req.MediaName == "" {
		return nil, status.Error(codes.InvalidArgument, "media name must be specified")
	}
	if req.DurationSecs <= 0 {
		return nil, status.Error(codes.InvalidArgument, "duration must be greater than 0")
	}
	if req.Privacy == "" {
		return nil, status.Error(codes.InvalidArgument, "privacy setting must be specified")
	}
	if len(req.MediaData) > 0 && req.MediaName == "" {
		return nil, status.Error(codes.InvalidArgument, "media name must be specified if media data is provided")
	}

	// Call the service method to create the reel
	reelID, err := h.FeedSvc.CreateReel(
		ctx,
		req.AuthorId,
		req.Caption,
		req.MediaData,
		req.MediaName,
		int(req.DurationSecs),
		req.Privacy,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create reel: %v", err)
	}

	return &feedpb.FeedResponse{
		ContentId: reelID,
		Message:   "Reel created successfully",
	}, nil
}

func (h *FeedHandlers) CreateStory(ctx context.Context, req *feedpb.CreateStoryRequest) (*feedpb.FeedResponse, error) {
	if req.AuthorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid author ID")
	}
	if req.MediaName == "" {
		return nil, status.Error(codes.InvalidArgument, "media name must be specified")
	}
	if req.DurationSecs <= 0 {
		return nil, status.Error(codes.InvalidArgument, "duration must be greater than 0")
	}
	if req.MediaType == "" {
		return nil, status.Error(codes.InvalidArgument, "media type must be specified")
	}

	// Call the service method to create the story
	storyID, err := h.FeedSvc.CreateStory(
		ctx,
		req.AuthorId,
		req.MediaData,
		req.MediaType,
		req.MediaName,
		int(req.DurationSecs),
		req.Privacy,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create story: %v", err)
	}

	return &feedpb.FeedResponse{
		ContentId: storyID,
		Message:   "Story created successfully",
	}, nil
}

func (h *FeedHandlers) ReactToContent(ctx context.Context, req *feedpb.ReactionRequest) (*feedpb.FeedResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid content ID")
	}
	if req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "reaction type must be specified")
	}

	// Call the service method to react to content
	err := h.FeedSvc.ReactToContent(ctx, req.UserId, req.ContentId, req.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to react to content: %v", err)
	}

	return &feedpb.FeedResponse{
		Message: "Reaction added successfully",
	}, nil
}

func (h *FeedHandlers) GetReactions(ctx context.Context, req *feedpb.ContentID) (*feedpb.ReactionList, error) {
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid content ID")
	}

	// Call the service method to get reactions
	reactions, err := h.FeedSvc.GetReactions(ctx, req.ContentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get reactions: %v", err)
	}

	// Convert reactions to protobuf format
	var pbReactions []*feedpb.Reaction
	for _, reaction := range reactions {
		pbReactions = append(pbReactions, &feedpb.Reaction{
			UserId:    reaction.UserID,
			ContentId: reaction.ContentID,
			Type:      reaction.Type,
		})
	}

	return &feedpb.ReactionList{
		Reactions: pbReactions,
	}, nil
}

func (h *FeedHandlers) DeleteReaction(ctx context.Context, req *feedpb.DeleteReactionRequest) (*feedpb.FeedResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid content ID")
	}

	// Call the service method to delete reaction
	err := h.FeedSvc.DeleteReaction(ctx, req.UserId, req.ContentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete reaction: %v", err)
	}

	return &feedpb.FeedResponse{
		Message: "Reaction deleted successfully",
	}, nil
}

func (h *FeedHandlers) GetMediaRef(ctx context.Context, req *feedpb.ContentID) (*feedpb.MediaResponse, error) {
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid media reference ID")
	}

	// Call the service method to get media reference
	mediaRef, err := h.FeedSvc.GetMediaRef(ctx, req.ContentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get media reference: %v", err)
	}

	return &feedpb.MediaResponse{

		MediaRefId: mediaRef.MediaRefID,
		FilePath:   mediaRef.FilePath,
	}, nil
}

func (h *FeedHandlers) GetContent(ctx context.Context, req *feedpb.ContentID) (*feedpb.FeedResponse, error) {
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid content ID")
	}

	// Call the service method to get content
	content, url, err := h.FeedSvc.GetContent(ctx, req.ContentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get content: %v", err)
	}

	return &feedpb.FeedResponse{
		ContentId: content.ContentID,
		MediaUrl:  url,
		Message:   *content.TextContent,
	}, nil
}

func (h *FeedHandlers) DeleteContent(ctx context.Context, req *feedpb.ContentID) (*feedpb.FeedResponse, error) {
	if req.ContentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid content ID")
	}

	// Call the service method to delete content
	err := h.FeedSvc.DeleteContent(ctx, req.ContentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete content: %v", err)
	}

	return &feedpb.FeedResponse{
		Message: "Content deleted successfully",
	}, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *FeedHandlers) GetTimeline(ctx context.Context, req *feedpb.UserID) (*feedpb.TimelineResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Call the service method to get timeline
	contents, urls, err := h.FeedSvc.GetTimeline(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get timeline: %v", err)
	}

	var pbContents []*feedpb.TimelineContent

	for i, content := range contents {
		pbContents = append(pbContents, &feedpb.TimelineContent{
			ContentId: content.ContentID,
			AuthorId:  content.AuthorID,
			Type:      content.Type,
			Text:      safeString(content.TextContent),
			MediaUrl:  urls[i],
			Privacy:   content.Privacy,
			CreatedAt: content.CreatedAt.String(), // or .String()
		})
	}

	return &feedpb.TimelineResponse{
		Contents: pbContents,
	}, nil
}

func (h *FeedHandlers) GetUserContent(ctx context.Context, req *feedpb.GetUserContentRequest) (*feedpb.TimelineResponse, error) {
	if req.RequesterId <= 0 || req.TargetUserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user IDs")
	}

	// Call the service method to get user content
	contents, urls, err := h.FeedSvc.GetUserContent(ctx, req.RequesterId, req.TargetUserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user content: %v", err)
	}

	// Convert contents to protobuf format
	var pbContents []*feedpb.Content
	for i, content := range contents {
		pbContents = append(pbContents, &feedpb.Content{
			ContentId:   content.ContentID,
			TextContent: *content.TextContent,
			MediaUrl:    urls[i],
			AuthorId:    content.AuthorID,
			Type:        content.Type,
			Privacy:     content.Privacy,
			Timestamp:   content.UpdatedAt.String(),
		})
	}

	return &feedpb.TimelineResponse{
		Contents: pbContents,
	}, nil
}
