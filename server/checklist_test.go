package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestParseChecklistCommandWithDefaultTitle(t *testing.T) {
	t.Parallel()

	checklist, err := parseChecklistCommand("/checklist draft copy | review links | publish")
	require.NoError(t, err)
	require.Equal(t, defaultChecklistTitle, checklist.Title)
	require.Len(t, checklist.Items, 3)
	require.Equal(t, "draft copy", checklist.Items[0].Text)
	require.Equal(t, "item-3", checklist.Items[2].ID)
}

func TestParseChecklistCommandWithCustomTitle(t *testing.T) {
	t.Parallel()

	checklist, err := parseChecklistCommand("/checklist Launch prep :: QA signoff | update release notes")
	require.NoError(t, err)
	require.Equal(t, "Launch prep", checklist.Title)
	require.Len(t, checklist.Items, 2)
	require.Equal(t, "update release notes", checklist.Items[1].Text)
}

func TestToggleChecklistItem(t *testing.T) {
	t.Parallel()

	checklist := &Checklist{
		Title: "Ship it",
		Items: []ChecklistItem{
			{ID: "item-1", Text: "Run smoke test"},
		},
	}

	require.NoError(t, toggleChecklistItem(checklist, "item-1", "user-1", "alice"))
	require.True(t, checklist.Items[0].Checked)
	require.Equal(t, "user-1", checklist.Items[0].CheckedBy)
	require.Equal(t, "alice", checklist.Items[0].CheckedByUsername)

	require.NoError(t, toggleChecklistItem(checklist, "item-1", "user-2", "bob"))
	require.False(t, checklist.Items[0].Checked)
	require.Empty(t, checklist.Items[0].CheckedBy)
	require.Empty(t, checklist.Items[0].CheckedByUsername)
}

func TestChecklistToPropsValueRoundTrips(t *testing.T) {
	t.Parallel()

	original := &Checklist{
		Title:     "Launch prep",
		CreatorID: "user-123",
		UpdatedAt: 123456,
		Items: []ChecklistItem{
			{ID: "item-1", Text: "QA signoff", Checked: true, CheckedBy: "user-123", CheckedByUsername: "bill"},
			{ID: "item-2", Text: "Post release note"},
		},
	}

	post := &PostAdapter{Props: map[string]any{checklistPropsKey: checklistToPropsValue(original)}}
	checklist, err := checklistFromPost(post.toModelPost())
	require.NoError(t, err)
	require.Equal(t, original, checklist)
}

func TestChecklistFromMessage(t *testing.T) {
	t.Parallel()

	checklist, err := checklistFromMessage("Launch prep\n\n- QA signoff\n- Update docs\n1. Post release note", "user-123")
	require.NoError(t, err)
	require.Equal(t, "Launch prep", checklist.Title)
	require.Equal(t, "user-123", checklist.CreatorID)
	require.Len(t, checklist.Items, 3)
	require.Equal(t, "QA signoff", checklist.Items[0].Text)
	require.Equal(t, "Post release note", checklist.Items[2].Text)
}

func TestChecklistFromMessageRequiresListItems(t *testing.T) {
	t.Parallel()

	_, err := checklistFromMessage("Just a paragraph", "user-123")
	require.ErrorIs(t, err, errChecklistItemsMissing)
}

func TestChecklistFromTaskSyntaxMessage(t *testing.T) {
	t.Parallel()

	checklist, err := checklistFromMessage("Weekly prep\n- [ ] Draft outline\n- [x] Review screenshots", "user-123")
	require.NoError(t, err)
	require.Equal(t, "Weekly prep", checklist.Title)
	require.Len(t, checklist.Items, 2)
	require.False(t, checklist.Items[0].Checked)
	require.True(t, checklist.Items[1].Checked)
	require.Equal(t, "Review screenshots", checklist.Items[1].Text)
}

type PostAdapter struct {
	Props map[string]any
}

func (p *PostAdapter) toModelPost() *model.Post {
	return &model.Post{Type: checklistPostType, Props: p.Props}
}
