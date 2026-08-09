package ui

import "my-new-go/internal/models"

type sidebarRowKind int

const (
	rowTree sidebarRowKind = iota
	rowHistory
)

// sidebarRow is one navigable line in the sidebar (tree node or history entry).
type sidebarRow struct {
	kind    sidebarRowKind
	path    []int // into Collection.Items for tree rows
	depth   int
	histIdx int
	id      string
	folder  bool
}

func copyPath(path []int) []int {
	if len(path) == 0 {
		return nil
	}
	out := make([]int, len(path))
	copy(out, path)
	return out
}

func pathsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func itemAt(items []models.Item, path []int) *models.Item {
	if len(path) == 0 {
		return nil
	}
	var cur *models.Item
	for i, idx := range path {
		if i == 0 {
			if idx < 0 || idx >= len(items) {
				return nil
			}
			cur = &items[idx]
		} else {
			if cur == nil || idx < 0 || idx >= len(cur.Children) {
				return nil
			}
			cur = &cur.Children[idx]
		}
	}
	return cur
}

// itemAtMut returns a pointer into the collection tree (writable).
func itemAtMut(col *models.Collection, path []int) *models.Item {
	if col == nil || len(path) == 0 {
		return nil
	}
	return itemAt(col.Items, path)
}

func (m *Model) isExpanded(id string) bool {
	if m.expanded == nil {
		return true
	}
	v, ok := m.expanded[id]
	if !ok {
		return true
	}
	return v
}

func (m *Model) setExpanded(id string, open bool) {
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	m.expanded[id] = open
}

func (m *Model) toggleExpanded(id string) {
	m.setExpanded(id, !m.isExpanded(id))
}

// sidebarRows builds the flattened visible sidebar list for the active collection.
func (m Model) sidebarRows() []sidebarRow {
	var rows []sidebarRow
	col := m.activeCollection()
	if col != nil {
		m.flattenItems(col.Items, nil, 0, &rows)
	}
	for i := range m.ws.History {
		rows = append(rows, sidebarRow{
			kind:    rowHistory,
			histIdx: i,
			id:      m.ws.History[i].ID,
		})
	}
	return rows
}

func (m Model) flattenItems(items []models.Item, path []int, depth int, rows *[]sidebarRow) {
	for i := range items {
		p := append(copyPath(path), i)
		it := items[i]
		*rows = append(*rows, sidebarRow{
			kind:   rowTree,
			path:   p,
			depth:  depth,
			id:     it.ID,
			folder: it.Kind == models.ItemFolder,
		})
		if it.Kind == models.ItemFolder && m.isExpanded(it.ID) {
			m.flattenItems(it.Children, p, depth+1, rows)
		}
	}
}

// insertParentChildren returns the Children slice (or collection root Items) where
// new requests/folders should be appended, based on the current sidebar cursor.
func (m *Model) insertParentChildren() *[]models.Item {
	col := m.activeCollection()
	if col == nil {
		return nil
	}
	rows := m.sidebarRows()
	if m.sidebarCursor >= 0 && m.sidebarCursor < len(rows) {
		row := rows[m.sidebarCursor]
		if row.kind == rowTree {
			item := itemAtMut(col, row.path)
			if item == nil {
				return &col.Items
			}
			if item.Kind == models.ItemFolder {
				return &item.Children
			}
			// Parent of selected request.
			if len(row.path) <= 1 {
				return &col.Items
			}
			parent := itemAtMut(col, row.path[:len(row.path)-1])
			if parent != nil {
				return &parent.Children
			}
			return &col.Items
		}
	}
	// History row or empty: use selectedPath parent, else root.
	if len(m.selectedPath) > 0 {
		item := itemAtMut(col, m.selectedPath)
		if item != nil && item.Kind == models.ItemFolder {
			return &item.Children
		}
		if len(m.selectedPath) > 1 {
			parent := itemAtMut(col, m.selectedPath[:len(m.selectedPath)-1])
			if parent != nil {
				return &parent.Children
			}
		}
	}
	return &col.Items
}

func (m *Model) cursorRow() (sidebarRow, bool) {
	rows := m.sidebarRows()
	if m.sidebarCursor < 0 || m.sidebarCursor >= len(rows) {
		return sidebarRow{}, false
	}
	return rows[m.sidebarCursor], true
}

func (m *Model) ensureSidebarCursor() {
	rows := m.sidebarRows()
	if len(rows) == 0 {
		m.sidebarCursor = 0
		return
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	if m.sidebarCursor >= len(rows) {
		m.sidebarCursor = len(rows) - 1
	}
}

func (m *Model) selectPath(path []int) {
	m.selectedPath = copyPath(path)
	rows := m.sidebarRows()
	for i, r := range rows {
		if r.kind == rowTree && pathsEqual(r.path, path) {
			m.sidebarCursor = i
			return
		}
	}
}

func (m *Model) firstRequestPath(items []models.Item, prefix []int) []int {
	for i, it := range items {
		p := append(copyPath(prefix), i)
		if it.Kind == models.ItemRequest {
			return p
		}
		if it.Kind == models.ItemFolder {
			if found := m.firstRequestPath(it.Children, p); found != nil {
				return found
			}
		}
	}
	return nil
}

func deleteAtPath(items *[]models.Item, path []int) bool {
	if len(path) == 0 || items == nil {
		return false
	}
	if len(path) == 1 {
		idx := path[0]
		if idx < 0 || idx >= len(*items) {
			return false
		}
		*items = append((*items)[:idx], (*items)[idx+1:]...)
		return true
	}
	idx := path[0]
	if idx < 0 || idx >= len(*items) {
		return false
	}
	return deleteAtPath(&(*items)[idx].Children, path[1:])
}
