# Interactive Checklist Plugin

This Mattermost plugin adds a shared checklist post type. A user creates a checklist with a slash command, and anyone in the channel can tick items on or off directly inside the message.

## Usage

Create a checklist with:

```text
/checklist item one | item two | item three
```

Add an optional title with:

```text
/checklist Launch prep :: QA signoff | update docs | post release note
```

The plugin stores checklist state in the post props and also updates the markdown body so non-enhanced clients still see the current state.

## Project Layout

- `server/checklist.go`: slash command handling, checklist parsing, and toggle API.
- `server/api.go`: authenticated plugin routes.
- `webapp/src/components/checklist_post.tsx`: custom post renderer with interactive checkboxes.
- `plugin.json`: Mattermost manifest for the bundled server and webapp plugin.

## Building

The standard Mattermost plugin workflow still applies:

```bash
make dist
```

This repo expects both Go and Node to be installed. In this workspace, the webapp can be built locally, but the server build cannot be verified until Go is available.

### How do I build the plugin with unminified JavaScript?
Setting the `MM_DEBUG` environment variable will invoke the debug builds. The simplist way to do this is to simply include this variable in your calls to `make` (e.g. `make dist MM_DEBUG=1`).
