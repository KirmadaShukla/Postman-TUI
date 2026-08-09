package models

import "testing"

func TestMigrateCollectionFlatRequests(t *testing.T) {
	col := Collection{
		ID:   "c1",
		Name: "Default",
		Requests: []Request{
			{ID: "r1", Name: "A", Method: "GET", URL: "https://example.com/a"},
			{ID: "r2", Name: "B", Method: "POST", URL: "https://example.com/b"},
		},
	}
	MigrateCollection(&col)
	if len(col.Requests) != 0 {
		t.Fatalf("legacy requests should be cleared, got %d", len(col.Requests))
	}
	if len(col.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(col.Items))
	}
	if col.Items[0].Kind != ItemRequest || col.Items[0].Request == nil {
		t.Fatal("item 0 should be a request")
	}
	if col.Items[0].Request.Name != "A" || col.Items[1].Request.Name != "B" {
		t.Fatalf("unexpected request names: %#v %#v", col.Items[0], col.Items[1])
	}
}

func TestMigrateCollectionKeepsItems(t *testing.T) {
	col := Collection{
		ID:   "c1",
		Name: "Default",
		Items: []Item{
			NewFolderItem("Auth"),
		},
		Requests: []Request{
			{ID: "r1", Name: "legacy", Method: "GET", URL: "/"},
		},
	}
	MigrateCollection(&col)
	if len(col.Items) != 1 || col.Items[0].Kind != ItemFolder {
		t.Fatalf("should keep existing items, got %#v", col.Items)
	}
	if len(col.Requests) != 0 {
		t.Fatal("legacy requests should still be cleared")
	}
}

func TestCountRequestsNested(t *testing.T) {
	items := []Item{
		NewRequestItem(Request{ID: "1", Name: "r1"}),
		{
			ID:   "f1",
			Name: "folder",
			Kind: ItemFolder,
			Children: []Item{
				NewRequestItem(Request{ID: "2", Name: "r2"}),
				{
					ID:   "f2",
					Name: "nested",
					Kind: ItemFolder,
					Children: []Item{
						NewRequestItem(Request{ID: "3", Name: "r3"}),
					},
				},
			},
		},
	}
	if got := CountRequests(items); got != 3 {
		t.Fatalf("CountRequests=%d want 3", got)
	}
	if got := CountFolders(items); got != 2 {
		t.Fatalf("CountFolders=%d want 2", got)
	}
}
