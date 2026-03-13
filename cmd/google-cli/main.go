package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/nguyenvanduocit/google-mcp/services"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// outputMode controls whether results are printed as text (default) or JSON.
var outputMode string

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Top-level --env and --output flags must appear before the subcommand.
	// We parse them manually so subcommands can use their own flag.FlagSet.
	envFile := ""
	for i, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--env=") {
			envFile = strings.TrimPrefix(a, "--env=")
			os.Args = append(os.Args[:i+1], os.Args[i+2:]...)
			break
		}
		if a == "--env" && i+2 < len(os.Args) {
			envFile = os.Args[i+2]
			os.Args = append(os.Args[:i+1], os.Args[i+3:]...)
			break
		}
	}
	for i, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--output=") {
			outputMode = strings.TrimPrefix(a, "--output=")
			os.Args = append(os.Args[:i+1], os.Args[i+2:]...)
			break
		}
		if a == "--output" && i+2 < len(os.Args) {
			outputMode = os.Args[i+2]
			os.Args = append(os.Args[:i+1], os.Args[i+3:]...)
			break
		}
	}

	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Fprintf(os.Stderr, "failed to load env file %s: %v\n", envFile, err)
			os.Exit(1)
		}
	}

	if outputMode == "" {
		outputMode = "text"
	}
	if outputMode != "text" && outputMode != "json" {
		fmt.Fprintf(os.Stderr, "--output must be 'text' or 'json'\n")
		os.Exit(1)
	}

	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	// Calendar
	case "calendar-event":
		runCalendarEvent(args)
	case "calendar-find-time-slot":
		runCalendarFindTimeSlot(args)
	case "calendar-get-busy-times":
		runCalendarGetBusyTimes(args)

	// Gmail
	case "gmail-search":
		runGmailSearch(args)
	case "gmail-read-email":
		runGmailReadEmail(args)
	case "gmail-reply-email":
		runGmailReplyEmail(args)
	case "gmail-move-to-spam":
		runGmailMoveToSpam(args)
	case "gmail-filter":
		runGmailFilter(args)
	case "gmail-label":
		runGmailLabel(args)

	// GChat
	case "gchat-list-spaces":
		runGChatListSpaces(args)
	case "gchat-send-message":
		runGChatSendMessage(args)
	case "gchat-list-users":
		runGChatListUsers(args)
	case "gchat-list-all-users":
		runGChatListUsers(args) // same as list-users
	case "gchat-list-messages":
		runGChatListMessages(args)
	case "gchat-get-thread-messages":
		runGChatGetThreadMessages(args)
	case "gchat-create-thread":
		runGChatCreateThread(args)
	case "gchat-archive-thread":
		runGChatArchiveThread(args)
	case "gchat-delete-thread":
		runGChatDeleteThread(args)
	case "gchat-get-user-info":
		runGChatGetUserInfo(args)

	// YouTube
	case "youtube-video":
		runYouTubeVideo(args)
	case "youtube-video-update":
		runYouTubeVideoUpdate(args)
	case "youtube-comments":
		runYouTubeComments(args)
	case "youtube-captions":
		runYouTubeCaptions(args)

	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", sub)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`google-cli - CLI for Google MCP tools

Usage: google-cli [--env FILE] [--output text|json] <command> [flags]

Global flags:
  --env FILE       Load environment variables from FILE (e.g. .env)
  --output FORMAT  Output format: text (default) or json

Required environment variables:
  GOOGLE_CREDENTIALS_FILE  Path to OAuth2 credentials JSON
  GOOGLE_TOKEN_FILE        Path to OAuth2 token JSON

Calendar commands:
  calendar-event           Manage calendar events (create/update/list/respond)
  calendar-find-time-slot  Find available time slots
  calendar-get-busy-times  Get busy time periods for users

Gmail commands:
  gmail-search             Search emails
  gmail-read-email         Read a specific email
  gmail-reply-email        Reply to an email
  gmail-move-to-spam       Move emails to spam
  gmail-filter             Manage Gmail filters (create/list/delete)
  gmail-label              Manage Gmail labels (list/delete)

Google Chat commands:
  gchat-list-spaces        List all Chat spaces
  gchat-send-message       Send a message to a space
  gchat-list-users         List users across spaces
  gchat-list-all-users     Alias for gchat-list-users
  gchat-list-messages      Get messages from a space
  gchat-get-thread-messages Get messages from a thread
  gchat-create-thread      Create a new Chat space
  gchat-archive-thread     Archive a Chat space
  gchat-delete-thread      Delete a Chat space
  gchat-get-user-info      Get info for a user by ID

YouTube commands:
  youtube-video            List or get YouTube videos
  youtube-video-update     Update video metadata
  youtube-comments         Manage video comments (list/post/reply)
  youtube-captions         Download video captions

Run 'google-cli <command> --help' for command-specific flags.
`)
}

// ── helpers ─────────────────────────────────────────────────────────────────

func printResult(v interface{}) {
	if outputMode == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			fatal("failed to encode JSON: %v", err)
		}
		return
	}
	// text: pretty-print as JSON for structured data anyway
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fatal("environment variable %s must be set (use --env to load a .env file)", name)
	}
	return v
}

func newCalendarService() *calendar.Service {
	tokenFile := requireEnv("GOOGLE_TOKEN_FILE")
	credFile := requireEnv("GOOGLE_CREDENTIALS_FILE")
	client := services.GoogleHttpClient(tokenFile, credFile)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		fatal("failed to create Calendar service: %v", err)
	}
	return srv
}

func newGmailService() *gmail.Service {
	tokenFile := requireEnv("GOOGLE_TOKEN_FILE")
	credFile := requireEnv("GOOGLE_CREDENTIALS_FILE")
	client := services.GoogleHttpClient(tokenFile, credFile)
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		fatal("failed to create Gmail service: %v", err)
	}
	return srv
}

func newChatService() *chat.Service {
	tokenFile := requireEnv("GOOGLE_TOKEN_FILE")
	credFile := requireEnv("GOOGLE_CREDENTIALS_FILE")
	client := services.GoogleHttpClient(tokenFile, credFile)
	srv, err := chat.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		fatal("failed to create Chat service: %v", err)
	}
	return srv
}

func newYouTubeService() *youtube.Service {
	tokenFile := requireEnv("GOOGLE_TOKEN_FILE")
	credFile := requireEnv("GOOGLE_CREDENTIALS_FILE")
	client := services.GoogleHttpClient(tokenFile, credFile)
	srv, err := youtube.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		fatal("failed to create YouTube service: %v", err)
	}
	return srv
}

// ── Calendar ─────────────────────────────────────────────────────────────────

func runCalendarEvent(args []string) {
	fs := flag.NewFlagSet("calendar-event", flag.ExitOnError)
	action := fs.String("action", "", "Action: create, update, list, respond (required)")
	eventID := fs.String("event-id", "", "Event ID (update/respond)")
	summary := fs.String("summary", "", "Event title")
	description := fs.String("description", "", "Event description")
	startTime := fs.String("start-time", "", "Start time RFC3339")
	endTime := fs.String("end-time", "", "End time RFC3339")
	attendees := fs.String("attendees", "", "Comma-separated attendee emails")
	timeMin := fs.String("time-min", "", "List: start time RFC3339 (default: now)")
	timeMax := fs.String("time-max", "", "List: end time RFC3339 (default: +7d)")
	maxResults := fs.Int64("max-results", 10, "List: max events")
	response := fs.String("response", "", "Respond: accepted|declined|tentative")
	_ = fs.String("env", "", "")    // accepted but handled globally
	_ = fs.String("output", "", "") // accepted but handled globally
	fs.Parse(args)

	if *action == "" {
		fatal("--action is required (create, update, list, respond)")
	}

	svc := newCalendarService()

	switch *action {
	case "create":
		if *summary == "" || *startTime == "" || *endTime == "" {
			fatal("--summary, --start-time, --end-time required for create")
		}
		st, err := time.Parse(time.RFC3339, *startTime)
		if err != nil {
			fatal("invalid --start-time: %v", err)
		}
		et, err := time.Parse(time.RFC3339, *endTime)
		if err != nil {
			fatal("invalid --end-time: %v", err)
		}
		ev := &calendar.Event{
			Summary:     *summary,
			Description: *description,
			Start:       &calendar.EventDateTime{DateTime: st.Format(time.RFC3339)},
			End:         &calendar.EventDateTime{DateTime: et.Format(time.RFC3339)},
		}
		if *attendees != "" {
			for _, email := range strings.Split(*attendees, ",") {
				ev.Attendees = append(ev.Attendees, &calendar.EventAttendee{Email: strings.TrimSpace(email)})
			}
		}
		created, err := svc.Events.Insert("primary", ev).Do()
		if err != nil {
			fatal("failed to create event: %v", err)
		}
		printResult(map[string]string{"id": created.Id, "status": "created"})

	case "update":
		if *eventID == "" {
			fatal("--event-id required for update")
		}
		ev, err := svc.Events.Get("primary", *eventID).Do()
		if err != nil {
			fatal("failed to get event: %v", err)
		}
		if *summary != "" {
			ev.Summary = *summary
		}
		if *description != "" {
			ev.Description = *description
		}
		if *startTime != "" {
			st, err := time.Parse(time.RFC3339, *startTime)
			if err != nil {
				fatal("invalid --start-time: %v", err)
			}
			ev.Start.DateTime = st.Format(time.RFC3339)
		}
		if *endTime != "" {
			et, err := time.Parse(time.RFC3339, *endTime)
			if err != nil {
				fatal("invalid --end-time: %v", err)
			}
			ev.End.DateTime = et.Format(time.RFC3339)
		}
		if *attendees != "" {
			ev.Attendees = nil
			for _, email := range strings.Split(*attendees, ",") {
				ev.Attendees = append(ev.Attendees, &calendar.EventAttendee{Email: strings.TrimSpace(email)})
			}
		}
		updated, err := svc.Events.Update("primary", *eventID, ev).Do()
		if err != nil {
			fatal("failed to update event: %v", err)
		}
		printResult(map[string]string{"id": updated.Id, "status": "updated"})

	case "list":
		tMin := *timeMin
		if tMin == "" {
			tMin = time.Now().Format(time.RFC3339)
		}
		tMax := *timeMax
		if tMax == "" {
			tMax = time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
		}
		events, err := svc.Events.List("primary").
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(tMin).
			TimeMax(tMax).
			MaxResults(*maxResults).
			OrderBy("startTime").
			Do()
		if err != nil {
			fatal("failed to list events: %v", err)
		}
		list := make([]map[string]interface{}, 0, len(events.Items))
		for _, item := range events.Items {
			start, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			end, _ := time.Parse(time.RFC3339, item.End.DateTime)
			info := map[string]interface{}{
				"id":      item.Id,
				"summary": item.Summary,
				"start":   start.Format("2006-01-02 15:04"),
				"end":     end.Format("2006-01-02 15:04"),
			}
			if item.Description != "" {
				info["description"] = item.Description
			}
			list = append(list, info)
		}
		printResult(map[string]interface{}{"count": len(list), "events": list})

	case "respond":
		if *eventID == "" || *response == "" {
			fatal("--event-id and --response required for respond")
		}
		ev, err := svc.Events.Get("primary", *eventID).Do()
		if err != nil {
			fatal("failed to get event: %v", err)
		}
		for _, a := range ev.Attendees {
			if a.Self {
				a.ResponseStatus = *response
				break
			}
		}
		_, err = svc.Events.Update("primary", *eventID, ev).Do()
		if err != nil {
			fatal("failed to respond to event: %v", err)
		}
		printResult(map[string]string{"id": *eventID, "response": *response, "status": "ok"})

	default:
		fatal("unknown action %q (create, update, list, respond)", *action)
	}
}

func runCalendarFindTimeSlot(args []string) {
	fs := flag.NewFlagSet("calendar-find-time-slot", flag.ExitOnError)
	guests := fs.String("guests", "", "Comma-separated guest emails")
	room := fs.String("room", "", "Room name filter")
	startDate := fs.String("start-date", "", "Start date RFC3339 (required)")
	endDate := fs.String("end-date", "", "End date RFC3339 (required)")
	durationMinutes := fs.Float64("duration-minutes", 30, "Meeting duration in minutes (required)")
	workStart := fs.String("working-hours-start", "09:00", "Working hours start (HH:MM)")
	workEnd := fs.String("working-hours-end", "17:00", "Working hours end (HH:MM)")
	maxResults := fs.Int("max-results", 5, "Max slots to return")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *startDate == "" || *endDate == "" {
		fatal("--start-date and --end-date are required")
	}

	svc := newCalendarService()

	sd, err := time.Parse(time.RFC3339, *startDate)
	if err != nil {
		fatal("invalid --start-date: %v", err)
	}
	ed, err := time.Parse(time.RFC3339, *endDate)
	if err != nil {
		fatal("invalid --end-date: %v", err)
	}

	calendarsToCheck := []string{"primary"}
	if *guests != "" {
		for _, g := range strings.Split(*guests, ",") {
			calendarsToCheck = append(calendarsToCheck, strings.TrimSpace(g))
		}
	}

	type timeSlot struct {
		Start time.Time
		End   time.Time
	}
	type busyEntry struct {
		Start      time.Time
		End        time.Time
		Summary    string
		Organizer  string
		CalendarID string
	}

	allBusy := make([]timeSlot, 0)
	busyDetails := make([]busyEntry, 0)

	for _, calID := range calendarsToCheck {
		events, err := svc.Events.List(calID).
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(sd.Format(time.RFC3339)).
			TimeMax(ed.Format(time.RFC3339)).
			OrderBy("startTime").
			Do()
		if err != nil {
			continue
		}
		for _, ev := range events.Items {
			if *room != "" && !strings.Contains(strings.ToLower(ev.Location), strings.ToLower(*room)) {
				continue
			}
			if ev.Start.DateTime != "" && ev.End.DateTime != "" {
				s, _ := time.Parse(time.RFC3339, ev.Start.DateTime)
				e, _ := time.Parse(time.RFC3339, ev.End.DateTime)
				allBusy = append(allBusy, timeSlot{s, e})
				org := ""
				if ev.Organizer != nil {
					org = ev.Organizer.Email
				}
				busyDetails = append(busyDetails, busyEntry{s, e, ev.Summary, org, calID})
			}
		}
	}

	// merge overlapping busy slots
	mergeBusy := func(slots []timeSlot) []timeSlot {
		if len(slots) == 0 {
			return slots
		}
		for i := 0; i < len(slots); i++ {
			for j := i + 1; j < len(slots); j++ {
				if slots[i].Start.After(slots[j].Start) {
					slots[i], slots[j] = slots[j], slots[i]
				}
			}
		}
		merged := []timeSlot{slots[0]}
		for _, s := range slots[1:] {
			last := &merged[len(merged)-1]
			if !s.Start.After(last.End) {
				if s.End.After(last.End) {
					last.End = s.End
				}
			} else {
				merged = append(merged, s)
			}
		}
		return merged
	}

	parseHHMM := func(t string) (int, int) {
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			return 9, 0
		}
		var h, m int
		fmt.Sscanf(parts[0], "%d", &h)
		fmt.Sscanf(parts[1], "%d", &m)
		return h, m
	}

	merged := mergeBusy(allBusy)
	dur := time.Duration(*durationMinutes) * time.Minute
	wsh, wsm := parseHHMM(*workStart)
	weh, wem := parseHHMM(*workEnd)

	available := make([]timeSlot, 0)
	cur := sd
	for cur.Before(ed) && len(available) < *maxResults {
		if cur.Weekday() == time.Saturday || cur.Weekday() == time.Sunday {
			cur = cur.AddDate(0, 0, 1)
			continue
		}
		dayStart := time.Date(cur.Year(), cur.Month(), cur.Day(), wsh, wsm, 0, 0, cur.Location())
		dayEnd := time.Date(cur.Year(), cur.Month(), cur.Day(), weh, wem, 0, 0, cur.Location())
		if dayStart.Before(sd) {
			dayStart = sd
		}
		if dayEnd.After(ed) {
			dayEnd = ed
		}
		ct := dayStart
		for ct.Add(dur).Before(dayEnd) || ct.Add(dur).Equal(dayEnd) {
			slotEnd := ct.Add(dur)
			ok := true
			for _, b := range merged {
				if ct.Before(b.End) && slotEnd.After(b.Start) {
					ok = false
					if b.End.After(ct) {
						ct = b.End
					}
					break
				}
			}
			if ok {
				available = append(available, timeSlot{ct, slotEnd})
				if len(available) >= *maxResults {
					break
				}
				ct = ct.Add(30 * time.Minute)
			}
		}
		cur = cur.AddDate(0, 0, 1)
	}

	slots := make([]map[string]string, 0, len(available))
	for _, s := range available {
		slots = append(slots, map[string]string{
			"start": s.Start.Format("2006-01-02 15:04"),
			"end":   s.End.Format("2006-01-02 15:04"),
			"day":   s.Start.Format("Monday"),
		})
	}
	busyOut := make([]map[string]string, 0, len(busyDetails))
	for _, b := range busyDetails {
		cal := b.CalendarID
		if cal == "primary" {
			cal = "Your calendar"
		}
		busyOut = append(busyOut, map[string]string{
			"start":     b.Start.Format("2006-01-02 15:04"),
			"end":       b.End.Format("2006-01-02 15:04"),
			"summary":   b.Summary,
			"organizer": b.Organizer,
			"calendar":  cal,
		})
	}
	printResult(map[string]interface{}{
		"duration_minutes": *durationMinutes,
		"available_slots":  slots,
		"busy_times":       busyOut,
	})
}

func runCalendarGetBusyTimes(args []string) {
	fs := flag.NewFlagSet("calendar-get-busy-times", flag.ExitOnError)
	users := fs.String("users", "", "Comma-separated user emails (empty = primary)")
	startDate := fs.String("start-date", "", "Start date RFC3339 (required)")
	endDate := fs.String("end-date", "", "End date RFC3339 (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *startDate == "" || *endDate == "" {
		fatal("--start-date and --end-date are required")
	}

	svc := newCalendarService()

	sd, err := time.Parse(time.RFC3339, *startDate)
	if err != nil {
		fatal("invalid --start-date: %v", err)
	}
	ed, err := time.Parse(time.RFC3339, *endDate)
	if err != nil {
		fatal("invalid --end-date: %v", err)
	}

	cals := []string{"primary"}
	if *users != "" {
		cals = nil
		for _, u := range strings.Split(*users, ",") {
			cals = append(cals, strings.TrimSpace(u))
		}
	}

	type busyEntry struct {
		Start      time.Time
		End        time.Time
		Summary    string
		Organizer  string
		CalendarID string
	}
	var details []busyEntry

	for _, calID := range cals {
		events, err := svc.Events.List(calID).
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(sd.Format(time.RFC3339)).
			TimeMax(ed.Format(time.RFC3339)).
			OrderBy("startTime").
			Do()
		if err != nil {
			details = append(details, busyEntry{Summary: fmt.Sprintf("Error: %v", err), CalendarID: calID})
			continue
		}
		for _, ev := range events.Items {
			if ev.Start.DateTime != "" && ev.End.DateTime != "" {
				s, _ := time.Parse(time.RFC3339, ev.Start.DateTime)
				e, _ := time.Parse(time.RFC3339, ev.End.DateTime)
				org := ""
				if ev.Organizer != nil {
					if ev.Organizer.DisplayName != "" {
						org = ev.Organizer.DisplayName
					} else {
						org = ev.Organizer.Email
					}
				}
				details = append(details, busyEntry{s, e, ev.Summary, org, calID})
			}
		}
	}

	busyOut := make([]map[string]interface{}, 0, len(details))
	for _, b := range details {
		busyOut = append(busyOut, map[string]interface{}{
			"start":            b.Start.Format("2006-01-02 15:04"),
			"end":              b.End.Format("2006-01-02 15:04"),
			"summary":          b.Summary,
			"organizer":        b.Organizer,
			"calendar":         b.CalendarID,
			"day":              b.Start.Format("Monday"),
			"duration_minutes": int(b.End.Sub(b.Start).Minutes()),
		})
	}
	printResult(map[string]interface{}{
		"period": map[string]string{
			"start": sd.Format("2006-01-02 15:04"),
			"end":   ed.Format("2006-01-02 15:04"),
		},
		"calendars_checked": cals,
		"total_busy_times":  len(details),
		"busy_times":        busyOut,
	})
}

// ── Gmail ────────────────────────────────────────────────────────────────────

func runGmailSearch(args []string) {
	fs := flag.NewFlagSet("gmail-search", flag.ExitOnError)
	query := fs.String("query", "", "Gmail search query (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *query == "" {
		fatal("--query is required")
	}

	svc := newGmailService()
	resp, err := svc.Users.Messages.List("me").Q(*query).MaxResults(10).Do()
	if err != nil {
		fatal("failed to search emails: %v", err)
	}

	emails := make([]map[string]interface{}, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		message, err := svc.Users.Messages.Get("me", msg.Id).Do()
		if err != nil {
			continue
		}
		info := map[string]interface{}{"id": msg.Id, "snippet": message.Snippet}
		for _, h := range message.Payload.Headers {
			switch h.Name {
			case "From":
				info["from"] = h.Value
			case "Subject":
				info["subject"] = h.Value
			case "Date":
				info["date"] = h.Value
			}
		}
		emails = append(emails, info)
	}
	printResult(map[string]interface{}{"count": len(emails), "emails": emails})
}

func runGmailReadEmail(args []string) {
	fs := flag.NewFlagSet("gmail-read-email", flag.ExitOnError)
	messageID := fs.String("message-id", "", "Email message ID (required)")
	includeAttachments := fs.Bool("include-attachments", false, "Include attachment info")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *messageID == "" {
		fatal("--message-id is required")
	}

	svc := newGmailService()
	message, err := svc.Users.Messages.Get("me", *messageID).Format("full").Do()
	if err != nil {
		fatal("failed to get email: %v", err)
	}

	result := map[string]interface{}{
		"id":      message.Id,
		"headers": map[string]string{},
		"body":    "",
	}
	for _, h := range message.Payload.Headers {
		switch h.Name {
		case "From", "To", "Cc", "Subject", "Date":
			result["headers"].(map[string]string)[h.Name] = h.Value
		}
	}
	result["body"] = extractGmailBody(message.Payload)

	if *includeAttachments && len(message.Payload.Parts) > 0 {
		atts := make([]map[string]interface{}, 0)
		for _, p := range message.Payload.Parts {
			if p.Filename != "" {
				atts = append(atts, map[string]interface{}{
					"filename": p.Filename,
					"size":     p.Body.Size,
				})
			}
		}
		if len(atts) > 0 {
			result["attachments"] = atts
		}
	}
	printResult(result)
}

func extractGmailBody(payload *gmail.MessagePart) string {
	if payload.MimeType == "text/plain" && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			return fmt.Sprintf("Error decoding body: %v", err)
		}
		return string(data)
	}
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				continue
			}
			return string(data)
		}
	}
	return "No readable text body found"
}

func runGmailReplyEmail(args []string) {
	fs := flag.NewFlagSet("gmail-reply-email", flag.ExitOnError)
	messageID := fs.String("message-id", "", "Email message ID (required)")
	replyText := fs.String("reply-text", "", "Reply text (required)")
	replyAll := fs.Bool("reply-all", false, "Reply to all recipients")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *messageID == "" || *replyText == "" {
		fatal("--message-id and --reply-text are required")
	}

	svc := newGmailService()
	orig, err := svc.Users.Messages.Get("me", *messageID).Format("metadata").Do()
	if err != nil {
		fatal("failed to get original email: %v", err)
	}

	var from, to, subject, references, msgIDHeader string
	for _, h := range orig.Payload.Headers {
		switch h.Name {
		case "From":
			to = h.Value
		case "To":
			from = h.Value
		case "Subject":
			subject = h.Value
			if !strings.HasPrefix(strings.ToLower(subject), "re:") {
				subject = "Re: " + subject
			}
		case "Message-ID":
			msgIDHeader = h.Value
			references = h.Value
		case "References":
			references = h.Value + " " + msgIDHeader
		}
	}

	recipients := []string{to}
	if *replyAll {
		for _, r := range strings.Split(from, ",") {
			r = strings.TrimSpace(r)
			if r != "" && !strings.Contains(r, "me@") {
				recipients = append(recipients, r)
			}
		}
	}

	var raw strings.Builder
	raw.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(recipients, ", ")))
	raw.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	raw.WriteString(fmt.Sprintf("References: %s\r\n", references))
	raw.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msgIDHeader))
	raw.WriteString("\r\n")
	raw.WriteString(*replyText)

	msg := &gmail.Message{Raw: base64.URLEncoding.EncodeToString([]byte(raw.String()))}
	_, err = svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		fatal("failed to send reply: %v", err)
	}
	printResult(map[string]string{"status": "sent"})
}

func runGmailMoveToSpam(args []string) {
	fs := flag.NewFlagSet("gmail-move-to-spam", flag.ExitOnError)
	messageIDs := fs.String("message-ids", "", "Comma-separated message IDs (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *messageIDs == "" {
		fatal("--message-ids is required")
	}

	svc := newGmailService()
	ids := strings.Split(*messageIDs, ",")
	for _, id := range ids {
		id = strings.TrimSpace(id)
		_, err := svc.Users.Messages.Modify("me", id, &gmail.ModifyMessageRequest{
			AddLabelIds: []string{"SPAM"},
		}).Do()
		if err != nil {
			fatal("failed to move %s to spam: %v", id, err)
		}
	}
	printResult(map[string]interface{}{"status": "moved", "count": len(ids)})
}

func runGmailFilter(args []string) {
	fs := flag.NewFlagSet("gmail-filter", flag.ExitOnError)
	action := fs.String("action", "", "Action: create, list, delete (required)")
	filterID := fs.String("filter-id", "", "Filter ID (delete)")
	from := fs.String("from", "", "From criteria (create)")
	to := fs.String("to", "", "To criteria (create)")
	subject := fs.String("subject", "", "Subject criteria (create)")
	query := fs.String("query", "", "Query criteria (create)")
	addLabel := fs.Bool("add-label", false, "Add label (create)")
	labelName := fs.String("label-name", "", "Label name (create, if --add-label)")
	markImportant := fs.Bool("mark-important", false, "Mark important (create)")
	markRead := fs.Bool("mark-read", false, "Mark read (create)")
	archive := fs.Bool("archive", false, "Archive (create)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *action == "" {
		fatal("--action is required (create, list, delete)")
	}

	svc := newGmailService()

	switch *action {
	case "create":
		criteria := &gmail.FilterCriteria{}
		if *from != "" {
			criteria.From = *from
		}
		if *to != "" {
			criteria.To = *to
		}
		if *subject != "" {
			criteria.Subject = *subject
		}
		if *query != "" {
			criteria.Query = *query
		}
		act := &gmail.FilterAction{}
		if *addLabel {
			if *labelName == "" {
				fatal("--label-name required when --add-label is set")
			}
			label, err := createOrGetGmailLabel(svc, *labelName)
			if err != nil {
				fatal("failed to get/create label: %v", err)
			}
			act.AddLabelIds = []string{label.Id}
		}
		if *markImportant {
			act.AddLabelIds = append(act.AddLabelIds, "IMPORTANT")
		}
		if *markRead {
			act.RemoveLabelIds = append(act.RemoveLabelIds, "UNREAD")
		}
		if *archive {
			act.RemoveLabelIds = append(act.RemoveLabelIds, "INBOX")
		}
		f := &gmail.Filter{Criteria: criteria, Action: act}
		result, err := svc.Users.Settings.Filters.Create("me", f).Do()
		if err != nil {
			fatal("failed to create filter: %v", err)
		}
		printResult(map[string]string{"id": result.Id, "status": "created"})

	case "list":
		filters, err := svc.Users.Settings.Filters.List("me").Do()
		if err != nil {
			fatal("failed to list filters: %v", err)
		}
		out := make([]map[string]interface{}, 0, len(filters.Filter))
		for _, f := range filters.Filter {
			info := map[string]interface{}{
				"id":       f.Id,
				"criteria": map[string]string{},
				"actions":  map[string]interface{}{},
			}
			if f.Criteria.From != "" {
				info["criteria"].(map[string]string)["from"] = f.Criteria.From
			}
			if f.Criteria.To != "" {
				info["criteria"].(map[string]string)["to"] = f.Criteria.To
			}
			if f.Criteria.Subject != "" {
				info["criteria"].(map[string]string)["subject"] = f.Criteria.Subject
			}
			if f.Criteria.Query != "" {
				info["criteria"].(map[string]string)["query"] = f.Criteria.Query
			}
			if len(f.Action.AddLabelIds) > 0 {
				info["actions"].(map[string]interface{})["addLabels"] = f.Action.AddLabelIds
			}
			if len(f.Action.RemoveLabelIds) > 0 {
				info["actions"].(map[string]interface{})["removeLabels"] = f.Action.RemoveLabelIds
			}
			out = append(out, info)
		}
		printResult(map[string]interface{}{"count": len(out), "filters": out})

	case "delete":
		if *filterID == "" {
			fatal("--filter-id required for delete")
		}
		err := svc.Users.Settings.Filters.Delete("me", *filterID).Do()
		if err != nil {
			fatal("failed to delete filter: %v", err)
		}
		printResult(map[string]string{"id": *filterID, "status": "deleted"})

	default:
		fatal("unknown action %q (create, list, delete)", *action)
	}
}

func createOrGetGmailLabel(svc *gmail.Service, name string) (*gmail.Label, error) {
	labels, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, err
	}
	for _, l := range labels.Labels {
		if l.Name == name {
			return l, nil
		}
	}
	return svc.Users.Labels.Create("me", &gmail.Label{
		Name:                  name,
		MessageListVisibility: "show",
		LabelListVisibility:   "labelShow",
	}).Do()
}

func runGmailLabel(args []string) {
	fs := flag.NewFlagSet("gmail-label", flag.ExitOnError)
	action := fs.String("action", "", "Action: list, delete (required)")
	labelID := fs.String("label-id", "", "Label ID (delete)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *action == "" {
		fatal("--action is required (list, delete)")
	}

	svc := newGmailService()

	switch *action {
	case "list":
		labels, err := svc.Users.Labels.List("me").Do()
		if err != nil {
			fatal("failed to list labels: %v", err)
		}
		sys := make([]map[string]interface{}, 0)
		usr := make([]map[string]interface{}, 0)
		for _, l := range labels.Labels {
			info := map[string]interface{}{"id": l.Id, "name": l.Name}
			if l.MessagesTotal > 0 {
				info["messagesTotal"] = l.MessagesTotal
			}
			if l.Type == "system" {
				sys = append(sys, info)
			} else if l.Type == "user" {
				usr = append(usr, info)
			}
		}
		printResult(map[string]interface{}{
			"count":        len(labels.Labels),
			"systemLabels": sys,
			"userLabels":   usr,
		})

	case "delete":
		if *labelID == "" {
			fatal("--label-id required for delete")
		}
		err := svc.Users.Labels.Delete("me", *labelID).Do()
		if err != nil {
			fatal("failed to delete label: %v", err)
		}
		printResult(map[string]string{"id": *labelID, "status": "deleted"})

	default:
		fatal("unknown action %q (list, delete)", *action)
	}
}

// ── GChat ────────────────────────────────────────────────────────────────────

func runGChatListSpaces(args []string) {
	fs := flag.NewFlagSet("gchat-list-spaces", flag.ExitOnError)
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	svc := newChatService()
	spaces, err := svc.Spaces.List().Do()
	if err != nil {
		fatal("failed to list spaces: %v", err)
	}
	result := make([]map[string]interface{}, 0, len(spaces.Spaces))
	for _, s := range spaces.Spaces {
		result = append(result, map[string]interface{}{
			"name":        s.Name,
			"displayName": s.DisplayName,
			"type":        s.Type,
		})
	}
	printResult(result)
}

func runGChatSendMessage(args []string) {
	fs := flag.NewFlagSet("gchat-send-message", flag.ExitOnError)
	spaceName := fs.String("space-name", "", "Space name, e.g. spaces/1234 (required)")
	message := fs.String("message", "", "Text to send (required)")
	threadName := fs.String("thread-name", "", "Thread name to reply to")
	useMarkdown := fs.Bool("use-markdown", false, "Format message as markdown")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *spaceName == "" || *message == "" {
		fatal("--space-name and --message are required")
	}

	svc := newChatService()
	msg := &chat.Message{Text: *message}
	if *useMarkdown {
		msg.FormattedText = *message
	}
	call := svc.Spaces.Messages.Create(*spaceName, msg)
	if *threadName != "" {
		call = call.ThreadKey(*threadName)
	}
	resp, err := call.Do()
	if err != nil {
		fatal("failed to send message: %v", err)
	}
	printResult(map[string]string{"name": resp.Name, "status": "sent"})
}

func runGChatListUsers(args []string) {
	fs := flag.NewFlagSet("gchat-list-users", flag.ExitOnError)
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	svc := newChatService()
	spaces, err := svc.Spaces.List().Do()
	if err != nil {
		fatal("failed to list spaces: %v", err)
	}

	userEmails := make(map[string]map[string]interface{})
	for _, space := range spaces.Spaces {
		members, err := svc.Spaces.Members.List(space.Name).
			PageSize(1000).
			ShowGroups(true).
			UseAdminAccess(true).
			Do()
		if err != nil {
			continue
		}
		for _, m := range members.Memberships {
			if m.Member == nil {
				continue
			}
			info := map[string]interface{}{
				"name":        m.Member.Name,
				"displayName": m.Member.DisplayName,
				"type":        m.Member.Type,
				"role":        m.Role,
			}
			email := ""
			if strings.HasPrefix(m.Member.Name, "users/") {
				part := strings.TrimPrefix(m.Member.Name, "users/")
				if strings.Contains(part, "@") {
					email = part
					info["email"] = email
				}
			}
			if email != "" {
				if existing, ok := userEmails[email]; ok {
					existing["spaces"] = append(existing["spaces"].([]string), space.Name)
				} else {
					info["spaces"] = []string{space.Name}
					userEmails[email] = info
				}
			}
		}
	}

	users := make([]map[string]interface{}, 0, len(userEmails))
	for _, u := range userEmails {
		u["spaceCount"] = len(u["spaces"].([]string))
		users = append(users, u)
	}
	printResult(map[string]interface{}{
		"totalUsers":  len(users),
		"totalSpaces": len(spaces.Spaces),
		"users":       users,
	})
}

func runGChatListMessages(args []string) {
	fs := flag.NewFlagSet("gchat-list-messages", flag.ExitOnError)
	spaceName := fs.String("space-name", "", "Space name (required)")
	pageSize := fs.Int64("page-size", 100, "Max messages")
	pageToken := fs.String("page-token", "", "Pagination token")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *spaceName == "" {
		fatal("--space-name is required")
	}

	svc := newChatService()
	call := svc.Spaces.Messages.List(*spaceName).
		OrderBy("createTime desc").
		PageSize(*pageSize)
	if *pageToken != "" {
		call = call.PageToken(*pageToken)
	}
	messages, err := call.Do()
	if err != nil {
		fatal("failed to get messages: %v", err)
	}

	msgs := make([]map[string]interface{}, 0, len(messages.Messages))
	for _, msg := range messages.Messages {
		info := map[string]interface{}{
			"name":       msg.Name,
			"sender":     msg.Sender,
			"createTime": msg.CreateTime,
			"text":       msg.Text,
			"thread":     msg.Thread,
		}
		if len(msg.Attachment) > 0 {
			atts := make([]map[string]interface{}, 0)
			for _, a := range msg.Attachment {
				atts = append(atts, map[string]interface{}{
					"name":         a.Name,
					"contentName":  a.ContentName,
					"contentType":  a.ContentType,
					"source":       a.Source,
					"thumbnailUri": a.ThumbnailUri,
					"downloadUri":  a.DownloadUri,
				})
			}
			info["attachments"] = atts
		}
		msgs = append(msgs, info)
	}
	printResult(map[string]interface{}{
		"messages":      msgs,
		"nextPageToken": messages.NextPageToken,
	})
}

func runGChatGetThreadMessages(args []string) {
	fs := flag.NewFlagSet("gchat-get-thread-messages", flag.ExitOnError)
	spaceName := fs.String("space-name", "", "Space name (required)")
	threadName := fs.String("thread-name", "", "Thread name (required)")
	pageSize := fs.Int64("page-size", 100, "Max messages")
	pageToken := fs.String("page-token", "", "Pagination token")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *spaceName == "" || *threadName == "" {
		fatal("--space-name and --thread-name are required")
	}

	svc := newChatService()
	call := svc.Spaces.Messages.List(*spaceName).
		OrderBy("createTime desc").
		PageSize(*pageSize).
		Filter(fmt.Sprintf("thread.name = %s", *threadName))
	if *pageToken != "" {
		call = call.PageToken(*pageToken)
	}
	messages, err := call.Do()
	if err != nil {
		fatal("failed to get thread messages: %v", err)
	}

	msgs := make([]map[string]interface{}, 0, len(messages.Messages))
	for _, msg := range messages.Messages {
		info := map[string]interface{}{
			"name":       msg.Name,
			"sender":     msg.Sender,
			"createTime": msg.CreateTime,
			"text":       msg.Text,
			"thread":     msg.Thread,
		}
		if len(msg.Attachment) > 0 {
			atts := make([]map[string]interface{}, 0)
			for _, a := range msg.Attachment {
				atts = append(atts, map[string]interface{}{
					"name":        a.Name,
					"contentName": a.ContentName,
					"contentType": a.ContentType,
				})
			}
			info["attachments"] = atts
		}
		msgs = append(msgs, info)
	}
	printResult(map[string]interface{}{
		"threadName":    *threadName,
		"messages":      msgs,
		"nextPageToken": messages.NextPageToken,
	})
}

func runGChatCreateThread(args []string) {
	fs := flag.NewFlagSet("gchat-create-thread", flag.ExitOnError)
	displayName := fs.String("display-name", "", "Space display name (required)")
	userEmails := fs.String("user-emails", "", "Comma-separated user emails (required)")
	initialMessage := fs.String("initial-message", "", "Optional initial message")
	externalAllowed := fs.Bool("external-user-allowed", false, "Allow external users")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *displayName == "" || *userEmails == "" {
		fatal("--display-name and --user-emails are required")
	}

	svc := newChatService()
	space := &chat.Space{
		DisplayName:         *displayName,
		Type:                "ROOM",
		SpaceType:           "SPACE",
		ExternalUserAllowed: *externalAllowed,
	}
	created, err := svc.Spaces.Create(space).Do()
	if err != nil {
		fatal("failed to create space: %v", err)
	}

	successful := []string{}
	failed := []string{}
	for _, email := range strings.Split(*userEmails, ",") {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		_, err := svc.Spaces.Members.Create(created.Name, &chat.Membership{
			Member: &chat.User{Name: fmt.Sprintf("users/%s", email), Type: "HUMAN"},
		}).Do()
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", email, err))
		} else {
			successful = append(successful, email)
		}
	}

	result := map[string]interface{}{
		"space": map[string]interface{}{
			"name":                created.Name,
			"displayName":         created.DisplayName,
			"type":                created.Type,
			"spaceType":           created.SpaceType,
			"externalUserAllowed": created.ExternalUserAllowed,
		},
		"members": map[string]interface{}{
			"successful": successful,
			"failed":     failed,
		},
	}

	if *initialMessage != "" {
		msg := &chat.Message{Text: *initialMessage}
		sent, err := svc.Spaces.Messages.Create(created.Name, msg).Do()
		if err == nil {
			result["initialMessageId"] = sent.Name
		}
	}
	printResult(result)
}

func runGChatArchiveThread(args []string) {
	fs := flag.NewFlagSet("gchat-archive-thread", flag.ExitOnError)
	spaceName := fs.String("space-name", "", "Space name (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *spaceName == "" {
		fatal("--space-name is required")
	}

	svc := newChatService()
	space, err := svc.Spaces.Get(*spaceName).Do()
	if err != nil {
		fatal("failed to get space: %v", err)
	}
	space.SpaceHistoryState = "HISTORY_ON"
	updated, err := svc.Spaces.Patch(*spaceName, space).UpdateMask("spaceHistoryState").Do()
	if err != nil {
		fatal("failed to archive space: %v", err)
	}
	printResult(map[string]interface{}{
		"name":              updated.Name,
		"displayName":       updated.DisplayName,
		"spaceHistoryState": updated.SpaceHistoryState,
		"archived":          true,
	})
}

func runGChatDeleteThread(args []string) {
	fs := flag.NewFlagSet("gchat-delete-thread", flag.ExitOnError)
	spaceName := fs.String("space-name", "", "Space name (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *spaceName == "" {
		fatal("--space-name is required")
	}

	svc := newChatService()
	_, err := svc.Spaces.Delete(*spaceName).Do()
	if err != nil {
		fatal("failed to delete space: %v", err)
	}
	printResult(map[string]interface{}{"spaceName": *spaceName, "deleted": true})
}

func runGChatGetUserInfo(args []string) {
	fs := flag.NewFlagSet("gchat-get-user-info", flag.ExitOnError)
	userID := fs.String("user-id", "", "User ID, e.g. users/123456789 (required)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *userID == "" {
		fatal("--user-id is required")
	}
	if !strings.HasPrefix(*userID, "users/") {
		fatal("--user-id must start with 'users/'")
	}

	svc := newChatService()
	spaces, err := svc.Spaces.List().Do()
	if err != nil {
		fatal("failed to list spaces: %v", err)
	}

	for _, space := range spaces.Spaces {
		members, err := svc.Spaces.Members.List(space.Name).
			PageSize(1000).
			ShowGroups(true).
			UseAdminAccess(true).
			Do()
		if err != nil {
			continue
		}
		for _, m := range members.Memberships {
			if m.Member != nil && m.Member.Name == *userID {
				printResult(map[string]interface{}{
					"name":        m.Member.Name,
					"displayName": m.Member.DisplayName,
					"type":        m.Member.Type,
				})
				return
			}
		}
	}
	fmt.Fprintf(os.Stderr, "user %s not found in accessible spaces\n", *userID)
	os.Exit(1)
}

// ── YouTube ──────────────────────────────────────────────────────────────────

func runYouTubeVideo(args []string) {
	fs := flag.NewFlagSet("youtube-video", flag.ExitOnError)
	action := fs.String("action", "", "Action: list, get (required)")
	videoID := fs.String("video-id", "", "Video ID (get action)")
	query := fs.String("query", "", "Search query (list action)")
	maxResults := fs.Int64("max-results", 10, "Max results (list action)")
	order := fs.String("order", "date", "Sort order (list action): date|rating|relevance|title|viewCount")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *action == "" {
		fatal("--action is required (list, get)")
	}

	svc := newYouTubeService()

	switch *action {
	case "list":
		call := svc.Search.List([]string{"snippet"}).
			ForMine(true).
			Type("video").
			MaxResults(*maxResults).
			Order(*order)
		if *query != "" {
			call = call.Q(*query)
		}
		resp, err := call.Do()
		if err != nil {
			fatal("failed to list videos: %v", err)
		}
		videos := make([]map[string]interface{}, 0, len(resp.Items))
		for _, item := range resp.Items {
			videos = append(videos, map[string]interface{}{
				"video_id":     item.Id.VideoId,
				"title":        item.Snippet.Title,
				"published_at": item.Snippet.PublishedAt,
				"description":  item.Snippet.Description,
			})
		}
		printResult(map[string]interface{}{"count": len(videos), "videos": videos})

	case "get":
		if *videoID == "" {
			fatal("--video-id required for 'get'")
		}
		resp, err := svc.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
			Id(*videoID).Do()
		if err != nil {
			fatal("failed to get video: %v", err)
		}
		if len(resp.Items) == 0 {
			fatal("video not found: %s", *videoID)
		}
		v := resp.Items[0]
		info := map[string]interface{}{
			"video_id":     v.Id,
			"title":        v.Snippet.Title,
			"description":  v.Snippet.Description,
			"channel":      v.Snippet.ChannelTitle,
			"published_at": v.Snippet.PublishedAt,
			"tags":         v.Snippet.Tags,
			"category_id":  v.Snippet.CategoryId,
		}
		if v.Statistics != nil {
			info["views"] = v.Statistics.ViewCount
			info["likes"] = v.Statistics.LikeCount
			info["comments"] = v.Statistics.CommentCount
		}
		if v.ContentDetails != nil {
			info["duration"] = v.ContentDetails.Duration
		}
		if v.Status != nil {
			info["privacy_status"] = v.Status.PrivacyStatus
			info["upload_status"] = v.Status.UploadStatus
		}
		printResult(info)

	default:
		fatal("unknown action %q (list, get)", *action)
	}
}

func runYouTubeVideoUpdate(args []string) {
	fs := flag.NewFlagSet("youtube-video-update", flag.ExitOnError)
	videoID := fs.String("video-id", "", "Video ID (required)")
	title := fs.String("title", "", "New title")
	description := fs.String("description", "", "New description")
	tags := fs.String("tags", "", "Comma-separated tags")
	categoryID := fs.String("category-id", "", "Category ID")
	privacyStatus := fs.String("privacy-status", "", "Privacy: public|unlisted|private")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *videoID == "" {
		fatal("--video-id is required")
	}

	needsSnippet := *title != "" || *description != "" || *tags != "" || *categoryID != ""
	needsStatus := *privacyStatus != ""
	if !needsSnippet && !needsStatus {
		fatal("provide at least one of: --title, --description, --tags, --category-id, --privacy-status")
	}

	svc := newYouTubeService()
	fetchParts := []string{}
	if needsSnippet {
		fetchParts = append(fetchParts, "snippet")
	}
	if needsStatus {
		fetchParts = append(fetchParts, "status")
	}

	resp, err := svc.Videos.List(fetchParts).Id(*videoID).Do()
	if err != nil {
		fatal("failed to get video: %v", err)
	}
	if len(resp.Items) == 0 {
		fatal("video not found: %s", *videoID)
	}
	v := resp.Items[0]

	if needsSnippet {
		if *title != "" {
			v.Snippet.Title = *title
		}
		if *description != "" {
			v.Snippet.Description = *description
		}
		if *tags != "" {
			tagList := strings.Split(*tags, ",")
			for i := range tagList {
				tagList[i] = strings.TrimSpace(tagList[i])
			}
			v.Snippet.Tags = tagList
		}
		if *categoryID != "" {
			v.Snippet.CategoryId = *categoryID
		}
	}
	if needsStatus {
		v.Status.PrivacyStatus = *privacyStatus
	}

	_, err = svc.Videos.Update(fetchParts, v).Do()
	if err != nil {
		fatal("failed to update video: %v", err)
	}
	printResult(map[string]string{"video_id": *videoID, "status": "updated"})
}

func runYouTubeComments(args []string) {
	fs := flag.NewFlagSet("youtube-comments", flag.ExitOnError)
	action := fs.String("action", "", "Action: list, post, reply (required)")
	videoID := fs.String("video-id", "", "Video ID (list/post)")
	commentID := fs.String("comment-id", "", "Comment ID (reply)")
	text := fs.String("text", "", "Comment text (post/reply)")
	maxResults := fs.Int64("max-results", 20, "Max comments (list)")
	order := fs.String("order", "time", "Sort: time|relevance (list)")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *action == "" {
		fatal("--action is required (list, post, reply)")
	}

	svc := newYouTubeService()

	switch *action {
	case "list":
		if *videoID == "" {
			fatal("--video-id required for list")
		}
		resp, err := svc.CommentThreads.List([]string{"snippet", "replies"}).
			VideoId(*videoID).
			MaxResults(*maxResults).
			Order(*order).
			TextFormat("plainText").
			Do()
		if err != nil {
			fatal("failed to list comments: %v", err)
		}
		comments := make([]map[string]interface{}, 0, len(resp.Items))
		for _, thread := range resp.Items {
			top := thread.Snippet.TopLevelComment
			info := map[string]interface{}{
				"comment_id":   top.Id,
				"author":       top.Snippet.AuthorDisplayName,
				"text":         top.Snippet.TextDisplay,
				"likes":        top.Snippet.LikeCount,
				"published_at": top.Snippet.PublishedAt,
				"reply_count":  thread.Snippet.TotalReplyCount,
			}
			if thread.Replies != nil && len(thread.Replies.Comments) > 0 {
				replies := make([]map[string]interface{}, 0)
				for _, r := range thread.Replies.Comments {
					replies = append(replies, map[string]interface{}{
						"comment_id":   r.Id,
						"author":       r.Snippet.AuthorDisplayName,
						"text":         r.Snippet.TextDisplay,
						"likes":        r.Snippet.LikeCount,
						"published_at": r.Snippet.PublishedAt,
					})
				}
				info["replies"] = replies
			}
			comments = append(comments, info)
		}
		printResult(map[string]interface{}{"count": len(comments), "comments": comments})

	case "post":
		if *videoID == "" || *text == "" {
			fatal("--video-id and --text required for post")
		}
		thread := &youtube.CommentThread{
			Snippet: &youtube.CommentThreadSnippet{
				VideoId: *videoID,
				TopLevelComment: &youtube.Comment{
					Snippet: &youtube.CommentSnippet{TextOriginal: *text},
				},
			},
		}
		resp, err := svc.CommentThreads.Insert([]string{"snippet"}, thread).Do()
		if err != nil {
			fatal("failed to post comment: %v", err)
		}
		printResult(map[string]string{"comment_id": resp.Id, "status": "posted"})

	case "reply":
		if *commentID == "" || *text == "" {
			fatal("--comment-id and --text required for reply")
		}
		comment := &youtube.Comment{
			Snippet: &youtube.CommentSnippet{
				ParentId:     *commentID,
				TextOriginal: *text,
			},
		}
		resp, err := svc.Comments.Insert([]string{"snippet"}, comment).Do()
		if err != nil {
			fatal("failed to reply: %v", err)
		}
		printResult(map[string]string{"comment_id": resp.Id, "status": "replied"})

	default:
		fatal("unknown action %q (list, post, reply)", *action)
	}
}

func runYouTubeCaptions(args []string) {
	fs := flag.NewFlagSet("youtube-captions", flag.ExitOnError)
	videoID := fs.String("video-id", "", "Video ID (required)")
	language := fs.String("language", "", "Language code, e.g. en")
	format := fs.String("format", "text", "Output format: text|srt|vtt")
	_ = fs.String("env", "", "")
	_ = fs.String("output", "", "")
	fs.Parse(args)

	if *videoID == "" {
		fatal("--video-id is required")
	}

	svc := newYouTubeService()
	captionResp, err := svc.Captions.List([]string{"id", "snippet"}, *videoID).Do()
	if err != nil {
		fatal("failed to list captions: %v", err)
	}
	if len(captionResp.Items) == 0 {
		fatal("no captions available for video %s", *videoID)
	}

	var captionID, captionLang string
	for _, c := range captionResp.Items {
		if *language != "" && c.Snippet.Language == *language {
			captionID = c.Id
			captionLang = c.Snippet.Language
			break
		}
		if captionID == "" {
			captionID = c.Id
			captionLang = c.Snippet.Language
		}
	}

	dlCall := svc.Captions.Download(captionID)
	switch *format {
	case "srt":
		dlCall = dlCall.Tfmt("srt")
	case "vtt":
		dlCall = dlCall.Tfmt("vtt")
	default:
		dlCall = dlCall.Tfmt("srt")
	}

	resp, err := dlCall.Download()
	if err != nil {
		fatal("failed to download captions: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fatal("failed to read caption data: %v", err)
	}

	content := string(body)
	if *format == "text" {
		content = stripSRT(content)
	}

	printResult(map[string]interface{}{
		"video_id": *videoID,
		"language": captionLang,
		"format":   *format,
		"content":  content,
	})
}

func stripSRT(srt string) string {
	lines := strings.Split(srt, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "-->") {
			continue
		}
		isNum := true
		for _, c := range line {
			if c < '0' || c > '9' {
				isNum = false
				break
			}
		}
		if isNum {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
