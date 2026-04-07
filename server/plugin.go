package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// router is the HTTP router for handling API requests.
	router *mux.Router

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.router = p.initRouter()

	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	return p.API.RegisterCommand(&model.Command{
		Trigger:          checklistCommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Create a collaborative checklist message",
		AutoCompleteHint: "[title ::] item one | item two | item three",
		AutocompleteData: model.NewAutocompleteData(
			checklistCommandTrigger,
			"[title ::] item one | item two | item three",
			"Create a shared checklist in the current channel",
		),
	})
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if len(fields) == 0 {
		return ephemeralResponse("Please provide a checklist definition."), nil
	}

	switch strings.TrimPrefix(fields[0], "/") {
	case checklistCommandTrigger:
		response, err := p.executeChecklistCommand(args)
		if err != nil {
			return nil, model.NewAppError(
				"ExecuteCommand",
				"plugin.command.execute_command.app_error",
				nil,
				err.Error(),
				http.StatusInternalServerError,
			)
		}

		return response, nil
	default:
		return ephemeralResponse(fmt.Sprintf("Unknown command: %s", args.Command)), nil
	}
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	if post == nil || post.Type != "" || strings.TrimSpace(post.Message) == "" {
		return post, ""
	}

	checklist, err := checklistFromMessage(post.Message, post.UserId)
	if err != nil {
		if errors.Is(err, errChecklistItemsMissing) {
			return post, ""
		}
		return post, err.Error()
	}

	hasTaskSyntax := false
	for _, line := range strings.Split(post.Message, "\n") {
		if _, _, ok := extractTaskChecklistItem(line); ok {
			hasTaskSyntax = true
			break
		}
	}
	if !hasTaskSyntax {
		return post, ""
	}

	applyChecklistToPost(post, checklist)
	return post, ""
}

func (p *Plugin) MessageWillBeUpdated(c *plugin.Context, newPost, oldPost *model.Post) (*model.Post, string) {
	if newPost == nil || oldPost == nil || strings.TrimSpace(newPost.Message) == "" {
		return newPost, ""
	}

	if oldPost.Type != checklistPostType {
		return newPost, ""
	}

	previousChecklist, err := checklistFromPost(oldPost)
	if err != nil {
		previousChecklist = nil
	}

	if propsChecklist, err := checklistFromPost(newPost); err == nil && checklistMatchesPostMessage(propsChecklist, newPost.Message) {
		propsChecklist.CreatorID = oldPost.UserId
		propsChecklist.UpdatedAt = model.GetMillis()
		applyChecklistToPost(newPost, propsChecklist)
		return newPost, ""
	}

	updatedChecklist, err := checklistFromMessage(newPost.Message, oldPost.UserId)
	if err != nil {
		return newPost, err.Error()
	}

	updatedChecklist = mergeChecklistState(previousChecklist, updatedChecklist)
	updatedChecklist.CreatorID = oldPost.UserId
	updatedChecklist.UpdatedAt = model.GetMillis()
	applyChecklistToPost(newPost, updatedChecklist)
	return newPost, ""
}

func checklistMatchesPostMessage(checklist *Checklist, message string) bool {
	if checklist == nil {
		return false
	}

	return strings.TrimSpace(renderChecklistMarkdown(checklist)) == strings.TrimSpace(message)
}

func ephemeralResponse(text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
