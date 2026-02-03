package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nguyenvanduocit/google-kit/services"
	"github.com/nguyenvanduocit/google-kit/util"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"gopkg.in/yaml.v3"
)

var youtubeService = sync.OnceValue(func() *youtube.Service {
	ctx := context.Background()

	tokenFile := os.Getenv("GOOGLE_TOKEN_FILE")
	if tokenFile == "" {
		panic("GOOGLE_TOKEN_FILE environment variable must be set")
	}

	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if credentialsFile == "" {
		panic("GOOGLE_CREDENTIALS_FILE environment variable must be set")
	}

	client := services.GoogleHttpClient(tokenFile, credentialsFile)

	srv, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(fmt.Sprintf("failed to create YouTube service: %v", err))
	}

	return srv
})

func RegisterYouTubeTools(s *server.MCPServer) {
	videoTool := mcp.NewTool("youtube_video",
		mcp.WithDescription("List or get YouTube videos from authenticated user's channel"),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action to perform: list, get")),
		mcp.WithString("video_id", mcp.Description("Video ID (required for 'get' action)")),
		mcp.WithString("query", mcp.Description("Search query to filter videos (optional for 'list' action)")),
		mcp.WithNumber("max_results", mcp.Description("Maximum results to return (default: 10, list action)")),
		mcp.WithString("order", mcp.Description("Sort order: date, rating, relevance, title, viewCount (default: date, list action)")),
	)
	s.AddTool(videoTool, util.ErrorGuard(youtubeVideoHandler))

	videoUpdateTool := mcp.NewTool("youtube_video_update",
		mcp.WithDescription("Update metadata for a YouTube video"),
		mcp.WithString("video_id", mcp.Required(), mcp.Description("Video ID to update")),
		mcp.WithString("title", mcp.Description("New video title")),
		mcp.WithString("description", mcp.Description("New video description")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
		mcp.WithString("category_id", mcp.Description("YouTube category ID (e.g., '22' for People & Blogs)")),
		mcp.WithString("privacy_status", mcp.Description("Privacy status: public, unlisted, private")),
	)
	s.AddTool(videoUpdateTool, util.ErrorGuard(youtubeVideoUpdateHandler))

	commentsTool := mcp.NewTool("youtube_comments",
		mcp.WithDescription("Manage YouTube video comments - list, post, or reply"),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action to perform: list, post, reply")),
		mcp.WithString("video_id", mcp.Description("Video ID (required for list/post actions)")),
		mcp.WithString("comment_id", mcp.Description("Comment ID (required for reply action)")),
		mcp.WithString("text", mcp.Description("Comment text (required for post/reply actions)")),
		mcp.WithNumber("max_results", mcp.Description("Maximum comments to return (default: 20, list action)")),
		mcp.WithString("order", mcp.Description("Sort order: time, relevance (default: time, list action)")),
	)
	s.AddTool(commentsTool, util.ErrorGuard(youtubeCommentsHandler))

	captionsTool := mcp.NewTool("youtube_captions",
		mcp.WithDescription("Download captions/transcript from a YouTube video"),
		mcp.WithString("video_id", mcp.Required(), mcp.Description("Video ID to get captions from")),
		mcp.WithString("language", mcp.Description("Language code (e.g., 'en', 'vi'). Default: first available")),
		mcp.WithString("format", mcp.Description("Output format: text (plain text, default), srt, vtt")),
	)
	s.AddTool(captionsTool, util.ErrorGuard(youtubeCaptionsHandler))
}

// Video handlers

func youtubeVideoHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	action, _ := arguments["action"].(string)

	switch action {
	case "list":
		return youtubeListVideosHandler(arguments)
	case "get":
		return youtubeGetVideoHandler(arguments)
	default:
		return mcp.NewToolResultError("Invalid action. Must be one of: list, get"), nil
	}
}

func youtubeListVideosHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, _ := arguments["query"].(string)
	maxResults, ok := arguments["max_results"].(float64)
	if !ok || maxResults <= 0 {
		maxResults = 10
	}
	order, _ := arguments["order"].(string)
	if order == "" {
		order = "date"
	}

	searchCall := youtubeService().Search.List([]string{"snippet"}).
		ForMine(true).
		Type("video").
		MaxResults(int64(maxResults)).
		Order(order)

	if query != "" {
		searchCall = searchCall.Q(query)
	}

	resp, err := searchCall.Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list videos: %v", err)), nil
	}

	videos := make([]map[string]interface{}, 0, len(resp.Items))
	for _, item := range resp.Items {
		videoInfo := map[string]interface{}{
			"video_id":     item.Id.VideoId,
			"title":        item.Snippet.Title,
			"published_at": item.Snippet.PublishedAt,
			"description":  item.Snippet.Description,
		}
		videos = append(videos, videoInfo)
	}

	result := map[string]interface{}{
		"count":  len(videos),
		"videos": videos,
	}

	yamlResult, err := yaml.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(yamlResult)), nil
}

func youtubeGetVideoHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	videoID, _ := arguments["video_id"].(string)
	if videoID == "" {
		return mcp.NewToolResultError("video_id is required for 'get' action"), nil
	}

	resp, err := youtubeService().Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
		Id(videoID).
		Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get video: %v", err)), nil
	}

	if len(resp.Items) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("video not found: %s", videoID)), nil
	}

	video := resp.Items[0]
	videoInfo := map[string]interface{}{
		"video_id":    video.Id,
		"title":       video.Snippet.Title,
		"description": video.Snippet.Description,
		"channel":     video.Snippet.ChannelTitle,
		"published_at": video.Snippet.PublishedAt,
		"tags":        video.Snippet.Tags,
		"category_id": video.Snippet.CategoryId,
	}

	if video.Statistics != nil {
		videoInfo["views"] = video.Statistics.ViewCount
		videoInfo["likes"] = video.Statistics.LikeCount
		videoInfo["comments"] = video.Statistics.CommentCount
	}

	if video.ContentDetails != nil {
		videoInfo["duration"] = video.ContentDetails.Duration
	}

	if video.Status != nil {
		videoInfo["privacy_status"] = video.Status.PrivacyStatus
		videoInfo["upload_status"] = video.Status.UploadStatus
	}

	yamlResult, err := yaml.Marshal(videoInfo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(yamlResult)), nil
}

// Video update handler

func youtubeVideoUpdateHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	videoID, _ := arguments["video_id"].(string)
	title, _ := arguments["title"].(string)
	description, _ := arguments["description"].(string)
	tagsStr, _ := arguments["tags"].(string)
	categoryID, _ := arguments["category_id"].(string)
	privacyStatus, _ := arguments["privacy_status"].(string)

	needsSnippet := title != "" || description != "" || tagsStr != "" || categoryID != ""
	needsStatus := privacyStatus != ""

	if !needsSnippet && !needsStatus {
		return mcp.NewToolResultError("no fields to update. Provide at least one of: title, description, tags, category_id, privacy_status"), nil
	}

	// Fetch only the parts we need to update
	fetchParts := []string{}
	if needsSnippet {
		fetchParts = append(fetchParts, "snippet")
	}
	if needsStatus {
		fetchParts = append(fetchParts, "status")
	}

	resp, err := youtubeService().Videos.List(fetchParts).
		Id(videoID).
		Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get video: %v", err)), nil
	}
	if len(resp.Items) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("video not found: %s", videoID)), nil
	}

	video := resp.Items[0]

	if needsSnippet {
		if title != "" {
			video.Snippet.Title = title
		}
		if description != "" {
			video.Snippet.Description = description
		}
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
			video.Snippet.Tags = tags
		}
		if categoryID != "" {
			video.Snippet.CategoryId = categoryID
		}
	}

	if needsStatus {
		video.Status.PrivacyStatus = privacyStatus
	}

	_, err = youtubeService().Videos.Update(fetchParts, video).Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update video: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully updated video %s", videoID)), nil
}

// Comments handlers

func youtubeCommentsHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	action, _ := arguments["action"].(string)

	switch action {
	case "list":
		return youtubeListCommentsHandler(arguments)
	case "post":
		return youtubePostCommentHandler(arguments)
	case "reply":
		return youtubeReplyCommentHandler(arguments)
	default:
		return mcp.NewToolResultError("Invalid action. Must be one of: list, post, reply"), nil
	}
}

func youtubeListCommentsHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	videoID, _ := arguments["video_id"].(string)
	if videoID == "" {
		return mcp.NewToolResultError("video_id is required for 'list' action"), nil
	}

	maxResults, ok := arguments["max_results"].(float64)
	if !ok || maxResults <= 0 {
		maxResults = 20
	}
	order, _ := arguments["order"].(string)
	if order == "" {
		order = "time"
	}

	resp, err := youtubeService().CommentThreads.List([]string{"snippet", "replies"}).
		VideoId(videoID).
		MaxResults(int64(maxResults)).
		Order(order).
		TextFormat("plainText").
		Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list comments: %v", err)), nil
	}

	comments := make([]map[string]interface{}, 0, len(resp.Items))
	for _, thread := range resp.Items {
		topComment := thread.Snippet.TopLevelComment
		commentInfo := map[string]interface{}{
			"comment_id":   topComment.Id,
			"author":       topComment.Snippet.AuthorDisplayName,
			"text":         topComment.Snippet.TextDisplay,
			"likes":        topComment.Snippet.LikeCount,
			"published_at": topComment.Snippet.PublishedAt,
			"reply_count":  thread.Snippet.TotalReplyCount,
		}

		if thread.Replies != nil && len(thread.Replies.Comments) > 0 {
			replies := make([]map[string]interface{}, 0, len(thread.Replies.Comments))
			for _, reply := range thread.Replies.Comments {
				replyInfo := map[string]interface{}{
					"comment_id":   reply.Id,
					"author":       reply.Snippet.AuthorDisplayName,
					"text":         reply.Snippet.TextDisplay,
					"likes":        reply.Snippet.LikeCount,
					"published_at": reply.Snippet.PublishedAt,
				}
				replies = append(replies, replyInfo)
			}
			commentInfo["replies"] = replies
		}

		comments = append(comments, commentInfo)
	}

	result := map[string]interface{}{
		"count":    len(comments),
		"comments": comments,
	}

	yamlResult, err := yaml.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(yamlResult)), nil
}

func youtubePostCommentHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	videoID, _ := arguments["video_id"].(string)
	if videoID == "" {
		return mcp.NewToolResultError("video_id is required for 'post' action"), nil
	}
	text, _ := arguments["text"].(string)
	if text == "" {
		return mcp.NewToolResultError("text is required for 'post' action"), nil
	}

	commentThread := &youtube.CommentThread{
		Snippet: &youtube.CommentThreadSnippet{
			VideoId: videoID,
			TopLevelComment: &youtube.Comment{
				Snippet: &youtube.CommentSnippet{
					TextOriginal: text,
				},
			},
		},
	}

	resp, err := youtubeService().CommentThreads.Insert([]string{"snippet"}, commentThread).Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to post comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Comment posted successfully. Comment ID: %s", resp.Id)), nil
}

func youtubeReplyCommentHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	commentID, _ := arguments["comment_id"].(string)
	if commentID == "" {
		return mcp.NewToolResultError("comment_id is required for 'reply' action"), nil
	}
	text, _ := arguments["text"].(string)
	if text == "" {
		return mcp.NewToolResultError("text is required for 'reply' action"), nil
	}

	comment := &youtube.Comment{
		Snippet: &youtube.CommentSnippet{
			ParentId:     commentID,
			TextOriginal: text,
		},
	}

	resp, err := youtubeService().Comments.Insert([]string{"snippet"}, comment).Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reply to comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Reply posted successfully. Comment ID: %s", resp.Id)), nil
}

// Captions handler

func youtubeCaptionsHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	videoID, _ := arguments["video_id"].(string)
	language, _ := arguments["language"].(string)
	format, _ := arguments["format"].(string)
	if format == "" {
		format = "text"
	}

	// List available caption tracks
	captionResp, err := youtubeService().Captions.List([]string{"id", "snippet"}, videoID).Do()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list captions: %v", err)), nil
	}

	if len(captionResp.Items) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("no captions available for video: %s", videoID)), nil
	}

	// Find the right caption track
	var captionID string
	var captionLang string
	for _, caption := range captionResp.Items {
		if language != "" && caption.Snippet.Language == language {
			captionID = caption.Id
			captionLang = caption.Snippet.Language
			break
		}
		if captionID == "" {
			captionID = caption.Id
			captionLang = caption.Snippet.Language
		}
	}

	// Download the caption
	downloadCall := youtubeService().Captions.Download(captionID)

	// Set format for download
	switch format {
	case "srt":
		downloadCall = downloadCall.Tfmt("srt")
	case "vtt":
		downloadCall = downloadCall.Tfmt("vtt")
	default:
		downloadCall = downloadCall.Tfmt("srt") // download as SRT, then strip timestamps
	}

	resp, err := downloadCall.Download()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to download captions: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read caption data: %v", err)), nil
	}

	content := string(body)

	// For plain text format, strip SRT formatting
	if format == "text" {
		content = stripSRTFormatting(content)
	}

	result := map[string]interface{}{
		"video_id": videoID,
		"language": captionLang,
		"format":   format,
		"content":  content,
	}

	yamlResult, err := yaml.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(yamlResult)), nil
}

// stripSRTFormatting removes SRT sequence numbers and timestamps, returning plain text
func stripSRTFormatting(srt string) string {
	lines := strings.Split(srt, "\n")
	var textLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, sequence numbers (pure digits), and timestamp lines (contain "-->")
		if line == "" {
			continue
		}
		if strings.Contains(line, "-->") {
			continue
		}
		// Skip pure numeric lines (sequence numbers)
		isNumber := true
		for _, c := range line {
			if c < '0' || c > '9' {
				isNumber = false
				break
			}
		}
		if isNumber {
			continue
		}
		textLines = append(textLines, line)
	}
	return strings.Join(textLines, "\n")
}
