package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	checklistCommandTrigger = "checklist"
	checklistPostType       = "custom_checklist"
	checklistPropsKey       = "checklist"
	defaultChecklistTitle   = "Checklist"
)

var errChecklistUsage = errors.New("usage: /checklist [title ::] item one | item two | item three")

type Checklist struct {
	Title     string          `json:"title"`
	CreatorID string          `json:"creator_id"`
	UpdatedAt int64           `json:"updated_at"`
	Items     []ChecklistItem `json:"items"`
}

type ChecklistItem struct {
	ID                string `json:"id"`
	Text              string `json:"text"`
	Checked           bool   `json:"checked"`
	CheckedAt         int64  `json:"checked_at,omitempty"`
	CheckedBy         string `json:"checked_by,omitempty"`
	CheckedByUsername string `json:"checked_by_username,omitempty"`
}

func (p *Plugin) executeChecklistCommand(args *model.CommandArgs) (*model.CommandResponse, error) {
	checklist, err := parseChecklistCommand(args.Command)
	if err != nil {
		return ephemeralResponse(err.Error()), nil
	}

	checklist.CreatorID = args.UserId
	checklist.UpdatedAt = model.GetMillis()

	post := &model.Post{
		UserId:    args.UserId,
		ChannelId: args.ChannelId,
		Type:      checklistPostType,
		Message:   renderChecklistMarkdown(checklist),
		Props: map[string]any{
			checklistPropsKey: checklist,
		},
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return nil, fmt.Errorf("create checklist post: %w", appErr)
	}

	return ephemeralResponse("Checklist posted."), nil
}

func parseChecklistCommand(command string) (*Checklist, error) {
	input := strings.TrimSpace(strings.TrimPrefix(command, "/"+checklistCommandTrigger))
	if input == "" || strings.EqualFold(input, "help") {
		return nil, errChecklistUsage
	}

	title := defaultChecklistTitle
	itemsInput := input

	if strings.Contains(input, "::") {
		parts := strings.SplitN(input, "::", 2)
		if customTitle := strings.TrimSpace(parts[0]); customTitle != "" {
			title = customTitle
		}
		itemsInput = parts[1]
	}

	rawItems := strings.Split(itemsInput, "|")
	items := make([]ChecklistItem, 0, len(rawItems))
	for index, rawItem := range rawItems {
		text := strings.TrimSpace(rawItem)
		if text == "" {
			continue
		}

		items = append(items, ChecklistItem{
			ID:   fmt.Sprintf("item-%d", index+1),
			Text: text,
		})
	}

	if len(items) == 0 {
		return nil, errChecklistUsage
	}

	return &Checklist{
		Title: title,
		Items: items,
	}, nil
}

func renderChecklistMarkdown(checklist *Checklist) string {
	var builder strings.Builder

	builder.WriteString("### ")
	builder.WriteString(checklist.Title)
	builder.WriteString("\n\n")

	for _, item := range checklist.Items {
		mark := "[ ]"
		if item.Checked {
			mark = "[x]"
		}

		builder.WriteString("- ")
		builder.WriteString(mark)
		builder.WriteString(" ")
		builder.WriteString(item.Text)

		if item.Checked && item.CheckedByUsername != "" {
			builder.WriteString(" _(checked by @")
			builder.WriteString(item.CheckedByUsername)
			builder.WriteString(")_")
		}

		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func checklistFromPost(post *model.Post) (*Checklist, error) {
	if post == nil || post.Type != checklistPostType {
		return nil, errors.New("post is not a checklist")
	}

	rawChecklist, ok := post.GetProps()[checklistPropsKey]
	if !ok {
		return nil, errors.New("checklist payload missing from post props")
	}

	bytes, err := json.Marshal(rawChecklist)
	if err != nil {
		return nil, fmt.Errorf("marshal checklist: %w", err)
	}

	var checklist Checklist
	if err := json.Unmarshal(bytes, &checklist); err != nil {
		return nil, fmt.Errorf("unmarshal checklist: %w", err)
	}

	return &checklist, nil
}

func applyChecklistToPost(post *model.Post, checklist *Checklist) {
	if post.Props == nil {
		post.Props = map[string]any{}
	}

	post.Type = checklistPostType
	post.Message = renderChecklistMarkdown(checklist)
	post.Props[checklistPropsKey] = checklist
}

func toggleChecklistItem(checklist *Checklist, itemID, userID, username string) error {
	for index := range checklist.Items {
		item := &checklist.Items[index]
		if item.ID != itemID {
			continue
		}

		if item.Checked {
			item.Checked = false
			item.CheckedAt = 0
			item.CheckedBy = ""
			item.CheckedByUsername = ""
		} else {
			item.Checked = true
			item.CheckedAt = model.GetMillis()
			item.CheckedBy = userID
			item.CheckedByUsername = username
		}

		return nil
	}

	return fmt.Errorf("checklist item %q not found", itemID)
}

func (p *Plugin) handleToggleChecklistItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "not authorized")
		return
	}

	vars := mux.Vars(r)
	postID := vars["postID"]
	itemID := vars["itemID"]
	if postID == "" || itemID == "" {
		writeAPIError(w, http.StatusBadRequest, "postID and itemID are required")
		return
	}

	post, appErr := p.API.GetPost(postID)
	if appErr != nil {
		writeAPIError(w, http.StatusNotFound, "checklist post not found")
		return
	}

	checklist, err := checklistFromPost(post)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	username := ""
	user, appErr := p.API.GetUser(userID)
	if appErr == nil && user != nil {
		username = user.Username
	}

	if err := toggleChecklistItem(checklist, itemID, userID, username); err != nil {
		writeAPIError(w, http.StatusNotFound, err.Error())
		return
	}

	checklist.UpdatedAt = model.GetMillis()
	applyChecklistToPost(post, checklist)

	updatedPost, appErr := p.API.UpdatePost(post)
	if appErr != nil {
		writeAPIError(w, http.StatusInternalServerError, "unable to update checklist post")
		return
	}

	updatedChecklist, err := checklistFromPost(updatedPost)
	if err != nil {
		updatedChecklist = checklist
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"checklist": updatedChecklist,
	})
}
