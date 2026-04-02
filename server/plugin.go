package main

import (
	"fmt"
	"net/http"
	"sync"
	"strings"

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

func ephemeralResponse(text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
