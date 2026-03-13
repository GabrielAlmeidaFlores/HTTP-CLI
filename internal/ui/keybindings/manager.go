package keybindings

import (
	"sort"
	"strings"
	"sync"

	"github.com/user/http-cli/internal/config"
)

type Binding struct {
	Action      string
	Keys        []string
	Description string
	Category    string
	Panel       string
	Visible     bool
	Priority    int
}

type Manager struct {
	mu       sync.RWMutex
	bindings []Binding
	byKey    map[string][]Binding
}

func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		bindings: make([]Binding, 0),
		byKey:    make(map[string][]Binding),
	}
	m.loadFromConfig(cfg)
	return m
}

func (m *Manager) loadFromConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.bindings = make([]Binding, 0)
	m.byKey = make(map[string][]Binding)

	panelPriority := map[string]int{
		"global":       0,
		"navigation":   1,
		"request_list": 10,
		"editor":       10,
		"response":     10,
	}

	for panel, actions := range cfg.Keybindings {
		priority := panelPriority[panel]
		for action, entry := range actions {
			b := Binding{
				Action:      action,
				Keys:        entry.Keys,
				Description: entry.Description,
				Category:    entry.Category,
				Panel:       panel,
				Visible:     entry.Visible,
				Priority:    priority,
			}
			m.bindings = append(m.bindings, b)
			for _, key := range entry.Keys {
				norm := normalizeKey(key)
				m.byKey[norm] = append(m.byKey[norm], b)
			}
		}
	}
}

func (m *Manager) Reload(cfg *config.Config) {
	m.loadFromConfig(cfg)
}

func (m *Manager) Resolve(key, panel string) (Binding, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	norm := normalizeKey(key)
	candidates := m.byKey[norm]
	if len(candidates) == 0 {
		return Binding{}, false
	}

	sorted := make([]Binding, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		si := scoreBinding(sorted[i], panel)
		sj := scoreBinding(sorted[j], panel)
		return si > sj
	})

	for _, b := range sorted {
		if matchesPanel(b, panel) {
			return b, true
		}
	}

	return Binding{}, false
}

func (m *Manager) GetHints(panel string) []Binding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var hints []Binding
	seen := make(map[string]bool)

	for _, b := range m.bindings {
		if !b.Visible {
			continue
		}
		if !matchesPanel(b, panel) {
			continue
		}
		key := b.Action + b.Panel
		if seen[key] {
			continue
		}
		seen[key] = true
		hints = append(hints, b)
	}

	sort.Slice(hints, func(i, j int) bool {
		if hints[i].Category != hints[j].Category {
			return hints[i].Category < hints[j].Category
		}
		return hints[i].Priority > hints[j].Priority
	})

	return hints
}

func matchesPanel(b Binding, panel string) bool {
	return b.Panel == "global" || b.Panel == "navigation" || b.Panel == panel
}

func scoreBinding(b Binding, panel string) int {
	if b.Panel == panel {
		return 20
	}
	if b.Panel == "global" {
		return 5
	}
	if b.Panel == "navigation" {
		return 3
	}
	return 0
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
