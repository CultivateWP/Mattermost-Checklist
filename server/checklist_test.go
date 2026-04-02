package main

import (
	"testing"

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
