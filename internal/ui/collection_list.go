package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type colNodeKind int

const (
	colNodeCollection colNodeKind = iota
	colNodeFolder
	colNodeRequest
)

type colNode struct {
	kind       colNodeKind
	depth      int
	expandKey  string
	requestID  string
	collection *models.Collection
	label      string
}

type CollectionListModel struct {
	keybindMgr   *keybindings.Manager
	collections  []*models.Collection
	requestIndex map[string]*models.Request
	expanded     map[string]bool
	visible      []colNode
	selectedIdx  int
	scrollOffset int
	width        int
	height       int
}

func newCollectionListModel(km *keybindings.Manager) CollectionListModel {
	return CollectionListModel{
		keybindMgr:   km,
		collections:  make([]*models.Collection, 0),
		requestIndex: make(map[string]*models.Request),
		expanded:     make(map[string]bool),
		visible:      make([]colNode, 0),
	}
}

func (m *CollectionListModel) setCollections(cols []*models.Collection) {
	m.collections = cols
	m.rebuild()
}

func (m *CollectionListModel) setRequests(reqs []*models.Request) {
	m.requestIndex = make(map[string]*models.Request, len(reqs))
	for _, r := range reqs {
		m.requestIndex[r.ID] = r
	}
	m.rebuild()
}

func (m *CollectionListModel) rebuild() {
	m.visible = m.visible[:0]
	for _, col := range m.collections {
		expandKey := col.ID
		total := countCollectionRequests(col)
		label := fmt.Sprintf("%s (%d)", col.Name, total)
		m.visible = append(m.visible, colNode{
			kind:       colNodeCollection,
			depth:      0,
			expandKey:  expandKey,
			collection: col,
			label:      label,
		})
		if m.expanded[expandKey] {
			for _, rid := range col.RequestIDs {
				m.visible = append(m.visible, m.requestNode(rid, col, 1))
			}
			for i := range col.Folders {
				m.appendFolderNodes(col, &col.Folders[i], col.ID+"/", 1)
			}
		}
	}
	if m.selectedIdx >= len(m.visible) {
		m.selectedIdx = len(m.visible) - 1
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

func (m *CollectionListModel) appendFolderNodes(col *models.Collection, folder *models.Folder, parentKey string, depth int) {
	expandKey := parentKey + folder.ID
	total := countFolderRequests(folder)
	label := fmt.Sprintf("%s (%d)", folder.Name, total)
	m.visible = append(m.visible, colNode{
		kind:       colNodeFolder,
		depth:      depth,
		expandKey:  expandKey,
		collection: col,
		label:      label,
	})
	if m.expanded[expandKey] {
		for _, rid := range folder.RequestIDs {
			m.visible = append(m.visible, m.requestNode(rid, col, depth+1))
		}
		for i := range folder.Folders {
			m.appendFolderNodes(col, &folder.Folders[i], expandKey+"/", depth+1)
		}
	}
}

func (m *CollectionListModel) requestNode(rid string, col *models.Collection, depth int) colNode {
	label := rid
	if req, ok := m.requestIndex[rid]; ok {
		label = fmt.Sprintf("%-7s %s", string(req.Method), req.Name)
	}
	return colNode{
		kind:       colNodeRequest,
		depth:      depth,
		expandKey:  "",
		requestID:  rid,
		collection: col,
		label:      label,
	}
}

func countCollectionRequests(col *models.Collection) int {
	count := len(col.RequestIDs)
	for i := range col.Folders {
		count += countFolderRequests(&col.Folders[i])
	}
	return count
}

func countFolderRequests(f *models.Folder) int {
	count := len(f.RequestIDs)
	for i := range f.Folders {
		count += countFolderRequests(&f.Folders[i])
	}
	return count
}

func (m *CollectionListModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *CollectionListModel) moveDown() {
	if m.selectedIdx < len(m.visible)-1 {
		m.selectedIdx++
		m.ensureVisible()
	}
}

func (m *CollectionListModel) moveUp() {
	if m.selectedIdx > 0 {
		m.selectedIdx--
		m.ensureVisible()
	}
}

func (m *CollectionListModel) toggle() {
	if len(m.visible) == 0 || m.selectedIdx >= len(m.visible) {
		return
	}
	node := m.visible[m.selectedIdx]
	if node.expandKey == "" {
		return
	}
	m.expanded[node.expandKey] = !m.expanded[node.expandKey]
	m.rebuild()
	m.ensureVisible()
}

func (m *CollectionListModel) selectedNode() *colNode {
	if len(m.visible) == 0 || m.selectedIdx >= len(m.visible) {
		return nil
	}
	n := m.visible[m.selectedIdx]
	return &n
}

func (m *CollectionListModel) selectedCollection() *models.Collection {
	node := m.selectedNode()
	if node == nil {
		return nil
	}
	return node.collection
}

func (m *CollectionListModel) ensureVisible() {
	if m.selectedIdx < m.scrollOffset {
		m.scrollOffset = m.selectedIdx
	}
	visible := m.height - 2
	if visible < 1 {
		visible = 1
	}
	if m.selectedIdx >= m.scrollOffset+visible {
		m.scrollOffset = m.selectedIdx - visible + 1
	}
}

func (m *CollectionListModel) view(focused bool, theme config.ThemeConfig) string {
	var lines []string
	visible := m.height - 2
	if visible < 1 {
		visible = 1
	}
	end := m.scrollOffset + visible
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := m.scrollOffset; i < end; i++ {
		lines = append(lines, m.renderNode(m.visible[i], i == m.selectedIdx, theme))
	}
	if len(m.visible) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Dim)).
			Render("No collections\nPress n to create"))
	}

	return panelBorderStyle(focused, theme).
		Width(m.width).
		Height(m.height).
		Padding(0, 1).
		Render("Collections\n" + strings.Join(lines, "\n"))
}

func (m *CollectionListModel) renderNode(node colNode, selected bool, theme config.ThemeConfig) string {
	indent := strings.Repeat("  ", node.depth)

	var arrow string
	switch node.kind {
	case colNodeCollection, colNodeFolder:
		if m.expanded[node.expandKey] {
			arrow = "▾ "
		} else {
			arrow = "▸ "
		}
	default:
		arrow = "  "
	}

	label := node.label
	maxLen := m.width - len([]rune(indent)) - len([]rune(arrow)) - 6
	if maxLen > 0 && len([]rune(label)) > maxLen {
		label = string([]rune(label)[:maxLen-1]) + "…"
	}

	var textStyle lipgloss.Style
	switch node.kind {
	case colNodeCollection:
		textStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
	case colNodeFolder:
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Secondary))
	default:
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ValueFg))
	}

	rendered := indent + textStyle.Render(arrow+label)

	if selected {
		return lipgloss.NewStyle().
			Background(lipgloss.Color(theme.ListSelectBg)).
			Render("> " + rendered)
	}
	return "  " + rendered
}
