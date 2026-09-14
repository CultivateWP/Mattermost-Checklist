# Interactive Checklist Plugin

This Mattermost plugin turns checklist-style posts into shared, interactive checklists that everyone in the channel can update.

## Current usage

The plugin supports three ways to create a checklist.

### 1. Use the `/checklist` slash command

Create a checklist from a single command:

```text
/checklist item one | item two | item three
```

Add an optional title with `::`:

```text
/checklist Launch prep :: QA signoff | update docs | post release note
```

That creates a custom checklist post with:

- clickable checklist items in the Mattermost webapp
- a completion summary at the top
- shared state for the whole channel
- markdown fallback in the post body for clients that do not render the custom UI

### 2. Use the `/theme-checklist` slash command

Post the standard Theme Build checklist with all items initially unchecked:

```text
/theme-checklist
```

The command creates the same interactive, collaborative checklist post as `/checklist`.

### 3. Post a normal Mattermost task list

If someone posts a standard Mattermost markdown task list, the plugin automatically converts it into an interactive checklist post.

Example:

```md
### Launch prep

- [ ] QA signoff
- [ ] Update docs
- [ ] Post release note
```

Notes:

- the first non-checklist line becomes the checklist title
- supported task markers include `- [ ]`, `* [ ]`, `+ [ ]`, `- [x]`, `* [x]`, and `+ [x]`
- plain bullet lists like `- item one` are not auto-converted; they stay normal posts

## Editing behavior

Once a post is a checklist:

- anyone in the channel can toggle items on or off
- checked items record who checked them in the markdown fallback
- if the post is edited later, existing checked/unchecked state is preserved by matching items on their text
- new items are added unchecked, and removed items drop out of the checklist

## Installation

### Option 1: Install a built release or local bundle

1. Build the plugin bundle:

   ```bash
   make dist
   ```

2. Upload the generated bundle from `dist/` into Mattermost. The bundle name will look like:

   ```text
   dist/com.billerickson.mattermost-checklist-<version>.tar.gz
   ```

3. In Mattermost, go to **System Console → Plugin Management**.
4. Enable plugin uploads if your server requires it.
5. Upload the `.tar.gz` bundle and enable the plugin.

After activation, the `/checklist` and `/theme-checklist` slash commands are registered automatically.

### Option 2: Deploy directly to a development server

This repo includes Mattermost's `pluginctl` helper, so you can build and deploy in one step:

```bash
make deploy
```

`make deploy` uses Mattermost local mode when available. Otherwise, set one of these before running it:

- `MM_SERVICESETTINGS_SITEURL`
- `MM_ADMIN_TOKEN`

Or:

- `MM_SERVICESETTINGS_SITEURL`
- `MM_ADMIN_USERNAME`
- `MM_ADMIN_PASSWORD`

## Development

Standard Mattermost plugin targets still apply:

```bash
make dist
make deploy
make watch
```

To build with unminified JavaScript:

```bash
make dist MM_DEBUG=1
```

## Project layout

- `server/checklist.go`: checklist parsing, state merging, slash command handling, and toggle/convert handlers
- `server/api.go`: authenticated plugin routes
- `server/plugin.go`: plugin activation, slash command registration, and message hooks
- `webapp/src/components/checklist_post.tsx`: custom post renderer with interactive checklist items
- `plugin.json`: Mattermost plugin manifest
