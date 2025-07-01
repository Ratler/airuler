// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2025 Stefan Wold <ratler@stderr.eu>

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InteractiveItem represents a selectable item in the TUI
type InteractiveItem struct {
	DisplayText string
	ID          string
	Data        interface{} // Store any additional data needed
	IsInstalled bool
	IsSelected  bool
}

// InteractiveModel is a reusable TUI model for selection interfaces
type InteractiveModel struct {
	Title        string
	Items        []InteractiveItem
	Selected     map[int]bool
	Cursor       int
	Done         bool
	Cancelled    bool
	Instructions string
	Viewport     viewport.Model
	Ready        bool
	VisibleStart int
	OnSelect     func(selectedItems []InteractiveItem) error // Callback for when selection is confirmed
	Formatter    ItemFormatter                               // Custom item formatter
	HeaderFormat HeaderFormatter                             // Custom header formatter
}

// InteractiveConfig holds configuration for the interactive TUI
type InteractiveConfig struct {
	Title        string
	Instructions string
	Items        []InteractiveItem
	OnSelect     func(selectedItems []InteractiveItem) error
	Formatter    ItemFormatter // Custom item formatter
}

// NewInteractiveModel creates a new interactive selection model
func NewInteractiveModel(config InteractiveConfig) InteractiveModel {
	return InteractiveModel{
		Title:        config.Title,
		Items:        config.Items,
		Selected:     make(map[int]bool),
		Cursor:       0,
		Done:         false,
		Cancelled:    false,
		Instructions: config.Instructions,
		Ready:        false,
		VisibleStart: 0,
		OnSelect:     config.OnSelect,
		Formatter:    config.Formatter,
	}
}

func (m InteractiveModel) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 4 // title + header + separator + blank line
		footerHeight := 3 // instructions + counter + blank line

		if !m.Ready {
			// Initialize viewport with manual scroll disabled
			m.Viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.Viewport.KeyMap = viewport.KeyMap{} // Disable all built-in key bindings
			m.Ready = true
			m.updateViewportContent()
		} else {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - headerHeight - footerHeight
			m.updateViewportContent()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Cancelled = true
			return m, tea.Quit
		case "up", "k":
			newCursor := m.findPrevSelectableItem(m.Cursor)
			if newCursor != m.Cursor {
				m.Cursor = newCursor
				m.adjustViewportScrolling()
			}
		case "down", "j":
			newCursor := m.findNextSelectableItem(m.Cursor)
			if newCursor != m.Cursor {
				m.Cursor = newCursor
				m.adjustViewportScrolling()
			}
		case " ":
			// Toggle selection only if not a group header
			if !m.isGroupHeader(m.Cursor) {
				if m.Selected[m.Cursor] {
					delete(m.Selected, m.Cursor)
				} else {
					m.Selected[m.Cursor] = true
				}
			}
			// Update content but don't change scroll position
			m.updateViewportContent()
		case "enter":
			m.Done = true
			return m, tea.Quit
		}
	}

	// Update viewport (but we've disabled its key bindings)
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m InteractiveModel) View() string {
	if !m.Ready {
		return "Loading..."
	}

	// Build the complete view with fixed header, viewport content, and footer
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.Viewport.View(),
		m.renderFooter(),
	)
}

// Helper functions for navigation
func (m InteractiveModel) isGroupHeader(index int) bool {
	if index < 0 || index >= len(m.Items) {
		return false
	}
	return strings.HasPrefix(m.Items[index].DisplayText, "GROUP_HEADER:")
}

// findGroupStart finds the start of the group that contains the given item index
func (m InteractiveModel) findGroupStart(itemIndex int) int {
	// Scan backwards from the current item to find the group header
	for i := itemIndex; i >= 0; i-- {
		if m.isGroupHeader(i) {
			return i // Return the group header index
		}
	}
	// If no group header found, always start from the beginning
	// This ensures we never lose context at the top
	return 0
}

func (m InteractiveModel) findNextSelectableItem(current int) int {
	for i := current + 1; i < len(m.Items); i++ {
		if !m.isGroupHeader(i) {
			return i
		}
	}
	return current // Stay at current if no next selectable item
}

func (m InteractiveModel) findPrevSelectableItem(current int) int {
	for i := current - 1; i >= 0; i-- {
		if !m.isGroupHeader(i) {
			return i
		}
	}
	return current // Stay at current if no previous selectable item
}

// updateViewportContent updates the viewport content with all items
func (m *InteractiveModel) updateViewportContent() {
	if !m.Ready {
		return
	}

	content := m.renderAllItems()
	m.Viewport.SetContent(content)
}

// adjustViewportScrolling handles scrolling only when cursor reaches edges
func (m *InteractiveModel) adjustViewportScrolling() {
	if !m.Ready {
		return
	}

	// Always update content first
	m.updateViewportContent()

	// Calculate current cursor line and ensure it's visible in viewport
	cursorLine := m.calculateItemLine(m.Cursor)
	currentOffset := m.Viewport.YOffset
	viewportHeight := m.Viewport.Height

	// Ensure we have valid viewport dimensions
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Only scroll if cursor is actually outside the visible viewport area
	// Add a small buffer to prevent oscillation at edges
	visibleTop := currentOffset
	visibleBottom := currentOffset + viewportHeight - 1

	// Add buffer for top edge to prevent cursor from disappearing behind group headers
	if cursorLine < visibleTop || (cursorLine == visibleTop && currentOffset > 0) {
		// Cursor is above visible area - scroll up to show it
		// Find the group start to ensure we show the group header
		groupStart := m.findGroupStart(m.Cursor)
		groupStartLine := m.calculateItemLine(groupStart)

		// Use the group start line as the offset to show both header and cursor
		newOffset := groupStartLine
		if newOffset < 0 {
			newOffset = 0
		}
		m.Viewport.SetYOffset(newOffset)
	} else if cursorLine > visibleBottom {
		// Cursor is below visible area - scroll down to show it
		newOffset := cursorLine - viewportHeight + 1
		if newOffset < 0 {
			newOffset = 0
		}
		m.Viewport.SetYOffset(newOffset)
	}
}

// calculateItemLine calculates which line an item appears on
func (m InteractiveModel) calculateItemLine(itemIndex int) int {
	line := 0
	for i := 0; i < len(m.Items) && i <= itemIndex; i++ {
		if strings.HasPrefix(m.Items[i].DisplayText, "GROUP_HEADER:") {
			if i == itemIndex {
				// If cursor is somehow ON a group header (which shouldn't happen),
				// return the line of the header text (line 1 of the 3-line group)
				return line + 1
			}
			line += 3 // Group headers take 3 lines (blank + header + blank)
		} else {
			if i == itemIndex {
				// If cursor is on a regular item, return its line
				return line
			}
			line += 1 // Regular items take 1 line
		}
	}
	return line
}

func (m InteractiveModel) renderHeader() string {
	var s strings.Builder

	// Title - always visible at top
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")) // White

	s.WriteString(titleStyle.Render(m.Title))
	s.WriteString("\n")

	// Table header - always visible
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")). // White text
		Background(lipgloss.Color("238"))  // Gray background

	s.WriteString(
		headerStyle.Render(fmt.Sprintf("   %-3s %-8s %-25s %-8s %-10s", "SEL", "TARGET", "TEMPLATE", "MODE", "STATUS")),
	)
	s.WriteString("\n")

	// Separator line - always visible
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // Medium gray
	s.WriteString(separatorStyle.Render(strings.Repeat("─", 60)))
	s.WriteString("\n")

	return s.String()
}

// renderAllItems renders all items for the viewport content
func (m InteractiveModel) renderAllItems() string {
	var s strings.Builder

	// Styles for content
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("238"))
		// White on gray
	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
		// Light gray
	installedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
		// Dark gray
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)
		// White
	groupHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Background(lipgloss.Color("236"))
		// White on dark gray

	// Render all items - the viewport will handle the scrolling window
	for i, item := range m.Items {
		// Handle group headers
		if strings.HasPrefix(item.DisplayText, "GROUP_HEADER:") {
			groupName := strings.TrimPrefix(item.DisplayText, "GROUP_HEADER:")
			s.WriteString("\n")
			s.WriteString(groupHeaderStyle.Render(fmt.Sprintf("   %s", groupName)))
			s.WriteString("\n")
			continue
		}

		cursor := " "
		if i == m.Cursor {
			cursor = cursorStyle.Render("►")
		}

		checkbox := "☐"
		style := unselectedStyle
		if item.IsInstalled {
			checkbox = "✓"
			style = installedStyle
		} else if m.Selected[i] {
			checkbox = "☑"
			style = selectedStyle
		}

		// The actual item rendering will be customized per use case
		// For now, we'll use a generic format that can be overridden
		row := m.formatItemRow(item, cursor, checkbox)
		s.WriteString(style.Render(row))
		s.WriteString("\n")
	}

	return s.String()
}

// ItemFormatter defines how to format individual items
type ItemFormatter func(item InteractiveItem, cursor, checkbox string) string

// HeaderFormatter defines how to format the table header
type HeaderFormatter func() string

// formatItemRow formats an item using the custom formatter or a default
func (m InteractiveModel) formatItemRow(item InteractiveItem, cursor, checkbox string) string {
	if m.Formatter != nil {
		return m.Formatter(item, cursor, checkbox)
	}
	// Default formatting - can be customized
	return fmt.Sprintf("%s %s %s", cursor, checkbox, item.DisplayText)
}

func (m InteractiveModel) renderFooter() string {
	var s strings.Builder

	// Instructions - always visible at bottom
	s.WriteString("\n")
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")). // Light gray
		Italic(true)
	s.WriteString(instructionStyle.Render(m.Instructions))

	// Selection counter - always visible at bottom
	s.WriteString("\n")
	counterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")). // White
		Bold(true)
	selectedCount := len(m.Selected)
	// Count only selectable items (exclude group headers)
	selectableCount := 0
	for i := range m.Items {
		if !m.isGroupHeader(i) && !m.Items[i].IsInstalled {
			selectableCount++
		}
	}
	s.WriteString(counterStyle.Render(fmt.Sprintf("Selected: %d of %d available", selectedCount, selectableCount)))

	return s.String()
}

// GetSelectedItems returns the items that were selected
func (m InteractiveModel) GetSelectedItems() []InteractiveItem {
	var selected []InteractiveItem
	for i := range m.Selected {
		if !m.isGroupHeader(i) {
			selected = append(selected, m.Items[i])
		}
	}
	return selected
}

// RunInteractiveSelection runs the interactive TUI and returns the result
func RunInteractiveSelection(config InteractiveConfig) ([]InteractiveItem, bool, error) {
	model := NewInteractiveModel(config)

	// Set cursor to first selectable item
	model.Cursor = model.findNextSelectableItem(-1)
	if model.Cursor == -1 && len(model.Items) > 0 {
		model.Cursor = 0
	}

	// Run the interactive program
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, false, fmt.Errorf("interactive selection failed: %w", err)
	}

	// Extract results
	final := finalModel.(InteractiveModel)
	if final.Cancelled {
		return nil, true, nil // cancelled = true
	}

	selectedItems := final.GetSelectedItems()
	return selectedItems, false, nil // cancelled = false
}
