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

var (
	errChecklistUsage        = errors.New("usage: /checklist [title ::] item one | item two | item three")
	errChecklistItemsMissing = errors.New("no checklist items found in the post")
)

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
			checklistPropsKey: checklistToPropsValue(checklist),
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

func checklistFromMessage(message, creatorID string) (*Checklist, error) {
	lines := strings.Split(message, "\n")
	items := make([]ChecklistItem, 0)
	title := defaultChecklistTitle
	firstContentLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if firstContentLine == "" {
			firstContentLine = trimmed
		}

		if text, checked, ok := extractTaskChecklistItem(trimmed); ok {
			item := ChecklistItem{
				ID:      fmt.Sprintf("item-%d", len(items)+1),
				Text:    text,
				Checked: checked,
			}
			if checked {
				item.CheckedAt = model.GetMillis()
			}
			items = append(items, item)
			continue
		}

		text, ok := extractChecklistItem(trimmed)
		if !ok {
			continue
		}

		items = append(items, ChecklistItem{
			ID:   fmt.Sprintf("item-%d", len(items)+1),
			Text: text,
		})
	}

	if len(items) == 0 {
		return nil, errChecklistItemsMissing
	}

	if firstContentLine != "" {
		if _, _, ok := extractTaskChecklistItem(firstContentLine); !ok {
			if text, ok := extractChecklistItem(firstContentLine); !ok || text == "" {
				title = strings.TrimSpace(strings.TrimLeft(firstContentLine, "# "))
			}
		}
	}

	return &Checklist{
		Title:     title,
		CreatorID: creatorID,
		UpdatedAt: model.GetMillis(),
		Items:     items,
	}, nil
}

func extractTaskChecklistItem(line string) (string, bool, bool) {
	trimmed := strings.TrimSpace(line)
	for _, marker := range []string{"- [ ] ", "* [ ] ", "+ [ ] ", "- [x] ", "* [x] ", "+ [x] ", "- [X] ", "* [X] ", "+ [X] "} {
		if strings.HasPrefix(trimmed, marker) {
			text := normalizeChecklistItemText(strings.TrimSpace(strings.TrimPrefix(trimmed, marker)))
			if text == "" {
				return "", false, false
			}
			checked := strings.Contains(marker, "[x]") || strings.Contains(marker, "[X]")
			return text, checked, true
		}
	}

	return "", false, false
}

func normalizeChecklistItemText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	if annotationIndex := strings.LastIndex(trimmed, " _(checked by @"); annotationIndex >= 0 && strings.HasSuffix(trimmed, ")_") {
		trimmed = strings.TrimSpace(trimmed[:annotationIndex])
	}

	return trimmed
}

func extractChecklistItem(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}

	for _, marker := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(trimmed, marker) {
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			return text, text != ""
		}
	}

	periodIndex := strings.Index(trimmed, ". ")
	if periodIndex > 0 {
		prefix := trimmed[:periodIndex]
		if prefix != "" {
			isNumbered := true
			for _, r := range prefix {
				if r < '0' || r > '9' {
					isNumbered = false
					break
				}
			}
			if isNumbered {
				text := strings.TrimSpace(trimmed[periodIndex+2:])
				return text, text != ""
			}
		}
	}

	return "", false
}

func checklistToPropsValue(checklist *Checklist) map[string]any {
	bytes, err := json.Marshal(checklist)
	if err != nil {
		return map[string]any{}
	}

	var propsValue map[string]any
	if err := json.Unmarshal(bytes, &propsValue); err != nil {
		return map[string]any{}
	}

	return propsValue
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
	post.Props[checklistPropsKey] = checklistToPropsValue(checklist)
}

func mergeChecklistState(previous, next *Checklist) *Checklist {
	if next == nil {
		return nil
	}

	if previous == nil {
		return next
	}

	type preservedItem struct {
		item ChecklistItem
		used bool
	}

	preservedByText := make(map[string][]*preservedItem, len(previous.Items))
	for _, item := range previous.Items {
		itemCopy := item
		key := strings.ToLower(strings.TrimSpace(item.Text))
		preservedByText[key] = append(preservedByText[key], &preservedItem{item: itemCopy})
	}

	for index := range next.Items {
		key := strings.ToLower(strings.TrimSpace(next.Items[index].Text))
		for _, candidate := range preservedByText[key] {
			if candidate.used {
				continue
			}

			next.Items[index].Checked = candidate.item.Checked
			next.Items[index].CheckedAt = candidate.item.CheckedAt
			next.Items[index].CheckedBy = candidate.item.CheckedBy
			next.Items[index].CheckedByUsername = candidate.item.CheckedByUsername
			candidate.used = true
			break
		}
	}

	return next
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
	userID, _ := p.userIDFromRequest(r)

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
	if userID != "" {
		user, appErr := p.API.GetUser(userID)
		if appErr == nil && user != nil {
			username = user.Username
		}
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

func (p *Plugin) handleConvertPostToChecklist(w http.ResponseWriter, r *http.Request) {
	userID, err := p.userIDFromRequest(r)
	if err != nil || userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "not authorized")
		return
	}

	postID := mux.Vars(r)["postID"]
	if postID == "" {
		writeAPIError(w, http.StatusBadRequest, "postID is required")
		return
	}

	post, appErr := p.API.GetPost(postID)
	if appErr != nil {
		writeAPIError(w, http.StatusNotFound, "post not found")
		return
	}

	if post.Type == checklistPostType {
		writeAPIError(w, http.StatusBadRequest, "post is already a checklist")
		return
	}

	checklist, err := checklistFromMessage(post.Message, post.UserId)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	applyChecklistToPost(post, checklist)
	updatedPost, appErr := p.API.UpdatePost(post)
	if appErr != nil {
		writeAPIError(w, http.StatusInternalServerError, "unable to convert post to checklist")
		return
	}

	updatedChecklist, err := checklistFromPost(updatedPost)
	if err != nil {
		updatedChecklist = checklist
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"checklist": updatedChecklist,
		"post_id":    updatedPost.Id,
	})
}
