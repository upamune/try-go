package trypkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectorMode int

const (
	modeBrowse selectorMode = iota
	modeRename
	modeConfirmDelete
)

type selectorOptions struct {
	AndType    string
	AndExit    bool
	AndKeys    []string
	AndConfirm string
}

type selectorResult struct {
	Kind        string
	Path        string
	BasePath    string
	OldName     string
	NewName     string
	DeleteNames []string
}

type dirEntry struct {
	Name  string
	Path  string
	MTime time.Time
	Base  float64
}

type scoredEntry struct {
	Entry dirEntry
	Score float64
}

type selectorModel struct {
	basePath string
	all      []dirEntry
	items    []scoredEntry

	query       string
	queryCursor int
	cursor      int
	scroll      int
	width       int
	height      int

	mode selectorMode

	markedPaths map[string]struct{}

	renameOriginal string
	renameText     string
	renameCursor   int

	confirmText   string
	confirmCursor int

	status string
	result selectorResult
	quit   bool
	err    error
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	selStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Bold(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
)

func setNoColors(disable bool) {
	if !disable {
		// Use stderr for profile detection so shell wrappers that capture stdout
		// (e.g. `out=$(try exec ...)`) still keep colors in interactive TUI output.
		renderer := lipgloss.NewRenderer(os.Stderr)
		titleStyle = renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		dimStyle = renderer.NewStyle().Foreground(lipgloss.Color("242"))
		selStyle = renderer.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
		errorStyle = renderer.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		dangerStyle = renderer.NewStyle().Foreground(lipgloss.Color("204")).Bold(true)
		cursorStyle = renderer.NewStyle().Reverse(true)
		return
	}
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle = lipgloss.NewStyle()
	selStyle = lipgloss.NewStyle().Bold(true)
	errorStyle = lipgloss.NewStyle().Bold(true)
	dangerStyle = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
}

func runSelector(basePath, initialQuery string, opts selectorOptions) (selectorResult, error) {
	entries, err := loadDirs(basePath)
	if err != nil {
		return selectorResult{}, err
	}
	if opts.AndType != "" {
		initialQuery = opts.AndType
	}
	m := selectorModel{
		basePath:    basePath,
		all:         entries,
		query:       sanitizeName(initialQuery),
		queryCursor: utf8.RuneCountInString(sanitizeName(initialQuery)),
		markedPaths: map[string]struct{}{},
	}
	m.refresh()

	if opts.AndExit && len(opts.AndKeys) == 0 {
		fmt.Fprintln(os.Stderr, m.View())
		return selectorResult{}, errRenderOnly
	}
	if len(opts.AndKeys) > 0 {
		for _, s := range opts.AndKeys {
			msg := testKeyToMsg(s)
			next, _ := m.Update(msg)
			m = next.(selectorModel)
			if m.mode == modeConfirmDelete && opts.AndConfirm != "" {
				m.confirmText = opts.AndConfirm
				m.confirmCursor = utf8.RuneCountInString(m.confirmText)
			}
			if m.quit {
				return selectorResult{}, errCancelled
			}
			if m.result.Kind != "" {
				return m.result, nil
			}
		}
		return selectorResult{}, errCancelled
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	out, err := p.Run()
	if err != nil {
		return selectorResult{}, err
	}
	fm, ok := out.(selectorModel)
	if !ok {
		return selectorResult{}, fmt.Errorf("selector model cast failed")
	}
	if fm.err != nil {
		return selectorResult{}, fm.err
	}
	if fm.quit {
		return selectorResult{}, errCancelled
	}
	return fm.result, nil
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.refresh()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeRename:
			return m.updateRename(msg)
		case modeConfirmDelete:
			return m.updateDeleteConfirm(msg)
		default:
			return m.updateBrowse(msg)
		}
	}
	return m, nil
}

func (m selectorModel) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureScroll()
		return m, nil
	case "down", "j", "ctrl+n":
		if m.cursor < m.totalRows()-1 {
			m.cursor++
		}
		m.ensureScroll()
		return m, nil
	case "enter":
		if len(m.markedPaths) > 0 {
			m.mode = modeConfirmDelete
			m.confirmText, m.confirmCursor = "", 0
			m.status = ""
			return m, nil
		}
		if m.cursor < len(m.items) {
			m.result = selectorResult{Kind: "cd", Path: m.items[m.cursor].Entry.Path}
			return m, tea.Quit
		}
		q := sanitizeName(m.query)
		if q == "" {
			return m, nil
		}
		today := time.Now().Format("2006-01-02")
		dir := uniqueDirName(m.basePath, fmt.Sprintf("%s-%s", today, q))
		m.result = selectorResult{Kind: "mkdir", Path: filepath.Join(m.basePath, dir)}
		return m, tea.Quit
	case "ctrl+r":
		if m.cursor < len(m.items) {
			name := m.items[m.cursor].Entry.Name
			m.mode = modeRename
			m.renameOriginal = name
			m.renameText = name
			m.renameCursor = utf8.RuneCountInString(name)
			m.status = ""
		}
		return m, nil
	case "ctrl+d":
		if m.cursor < len(m.items) {
			path := m.items[m.cursor].Entry.Path
			if _, ok := m.markedPaths[path]; ok {
				delete(m.markedPaths, path)
			} else {
				m.markedPaths[path] = struct{}{}
			}
		}
		return m, nil
	case "ctrl+t":
		q := sanitizeName(m.query)
		if q == "" {
			q = "new"
		}
		today := time.Now().Format("2006-01-02")
		dir := uniqueDirName(m.basePath, fmt.Sprintf("%s-%s", today, q))
		m.result = selectorResult{Kind: "mkdir", Path: filepath.Join(m.basePath, dir)}
		return m, tea.Quit
	}

	if applyEditKey(&m.query, &m.queryCursor, msg) {
		m.query = sanitizeName(m.query)
		m.queryCursor = clamp(m.queryCursor, 0, utf8.RuneCountInString(m.query))
		m.refresh()
		return m, nil
	}
	return m, nil
}

func (m selectorModel) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.status = ""
		return m, nil
	case "enter":
		newName := sanitizeName(m.renameText)
		if newName == "" {
			m.status = "Name cannot be empty"
			return m, nil
		}
		if strings.Contains(newName, "/") {
			m.status = "Name cannot contain /"
			return m, nil
		}
		if newName == m.renameOriginal {
			m.mode = modeBrowse
			m.status = ""
			return m, nil
		}
		if _, err := os.Stat(filepath.Join(m.basePath, newName)); err == nil {
			m.status = "Directory exists: " + newName
			return m, nil
		}
		m.result = selectorResult{
			Kind:     "rename",
			BasePath: m.basePath,
			OldName:  m.renameOriginal,
			NewName:  newName,
		}
		return m, tea.Quit
	}
	if applyEditKey(&m.renameText, &m.renameCursor, msg) {
		m.renameText = sanitizeRename(m.renameText)
		m.renameCursor = clamp(m.renameCursor, 0, utf8.RuneCountInString(m.renameText))
		m.status = ""
	}
	return m, nil
}

func (m selectorModel) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.markedPaths = map[string]struct{}{}
		m.status = "Delete cancelled"
		return m, nil
	case "enter":
		if m.confirmText != "YES" {
			m.mode = modeBrowse
			m.markedPaths = map[string]struct{}{}
			m.status = "Delete cancelled"
			return m, nil
		}
		names := m.markedBaseNames()
		if len(names) == 0 {
			m.mode = modeBrowse
			return m, nil
		}
		m.result = selectorResult{
			Kind:        "delete",
			BasePath:    m.basePath,
			DeleteNames: names,
		}
		return m, tea.Quit
	}
	if applyEditKey(&m.confirmText, &m.confirmCursor, msg) {
		m.confirmCursor = clamp(m.confirmCursor, 0, utf8.RuneCountInString(m.confirmText))
	}
	return m, nil
}

func (m selectorModel) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	if m.mode == modeRename {
		return m.viewRename()
	}
	if m.mode == modeConfirmDelete {
		return m.viewDeleteConfirm()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("try - directory selector"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Search: "))
	b.WriteString(renderInput(m.query, m.queryCursor))
	b.WriteString("\n\n")

	if m.totalRows() == 0 {
		b.WriteString(dimStyle.Render("No directories. Type a name and press Enter to create."))
		b.WriteString("\n")
	}

	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		selected := i == m.cursor
		if i < len(m.items) {
			mark := "  "
			if selected {
				mark = selStyle.Render("→ ")
			}
			prefix := "📁 "
			if _, ok := m.markedPaths[m.items[i].Entry.Path]; ok {
				prefix = dangerStyle.Render("🗑 ")
			}
			name := renderName(m.items[i].Entry.Name, m.width-8)
			b.WriteString(mark + prefix + name + "\n")
			continue
		}
		if m.showCreate() {
			mark := "  "
			if selected {
				mark = selStyle.Render("→ ")
			}
			b.WriteString(mark + "📂 Create: " + m.previewCreateName() + "\n")
		}
	}

	b.WriteString("\n")
	if m.status != "" {
		b.WriteString(errorStyle.Render(m.status))
		b.WriteString("\n")
	}
	if len(m.markedPaths) > 0 {
		b.WriteString(dangerStyle.Render("Delete mode: Enter confirm / Ctrl-D toggle / Esc cancel"))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("↑/↓ navigate  Enter select/create  Ctrl-R rename  Ctrl-D delete  Esc cancel"))
	return b.String()
}

func (m selectorModel) viewRename() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Rename directory"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Current: "))
	b.WriteString(m.renameOriginal)
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("New:     "))
	b.WriteString(renderInput(m.renameText, m.renameCursor))
	b.WriteString("\n\n")
	if m.status != "" {
		b.WriteString(errorStyle.Render(m.status))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("Enter confirm  Esc cancel  Ctrl-A/E/B/F/K/W edit"))
	return b.String()
}

func (m selectorModel) viewDeleteConfirm() string {
	var b strings.Builder
	names := m.markedBaseNames()
	b.WriteString(dangerStyle.Render(fmt.Sprintf("Delete %d director%s", len(names), plural(len(names), "y", "ies"))))
	b.WriteString("\n\n")
	for _, n := range names {
		b.WriteString("🗑 ")
		b.WriteString(n)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Type YES to confirm: "))
	b.WriteString(renderInput(m.confirmText, m.confirmCursor))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Enter confirm  Esc cancel"))
	return b.String()
}

func (m *selectorModel) refresh() {
	m.items = filterEntries(m.all, strings.TrimSpace(m.query))
	maxRow := m.totalRows() - 1
	if maxRow < 0 {
		m.cursor, m.scroll = 0, 0
		return
	}
	if m.cursor > maxRow {
		m.cursor = maxRow
	}
	m.ensureScroll()
}

func (m *selectorModel) ensureScroll() {
	rows := m.visibleRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+rows {
		m.scroll = m.cursor - rows + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m selectorModel) visibleRange() (int, int) {
	rows := m.visibleRows()
	start := m.scroll
	end := minInt(m.totalRows(), start+rows)
	if start > end {
		start = end
	}
	return start, end
}

func (m selectorModel) markedBaseNames() []string {
	names := make([]string, 0, len(m.markedPaths))
	for path := range m.markedPaths {
		names = append(names, filepath.Base(path))
	}
	sort.Strings(names)
	return names
}

func (m selectorModel) showCreate() bool {
	return strings.TrimSpace(m.query) != ""
}

func (m selectorModel) previewCreateName() string {
	q := sanitizeName(m.query)
	return time.Now().Format("2006-01-02") + "-" + q
}

func (m selectorModel) totalRows() int {
	n := len(m.items)
	if m.showCreate() {
		n++
	}
	return n
}

func (m selectorModel) visibleRows() int {
	if m.height <= 0 {
		return 12
	}
	return maxInt(3, m.height-8)
}

func loadDirs(basePath string) ([]dirEntry, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		hours := now.Sub(info.ModTime()).Hours()
		base := 3.0 / sqrt(hours+1)
		if datePrefix(e.Name()) {
			base += 2.0
		}
		out = append(out, dirEntry{
			Name:  e.Name(),
			Path:  filepath.Join(basePath, e.Name()),
			MTime: info.ModTime(),
			Base:  base,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].MTime.After(out[j].MTime)
	})
	return out, nil
}

func filterEntries(all []dirEntry, query string) []scoredEntry {
	query = strings.ToLower(strings.TrimSpace(query))
	res := make([]scoredEntry, 0, len(all))
	for _, e := range all {
		score, ok := fuzzyScore(strings.ToLower(e.Name), query)
		if !ok {
			continue
		}
		res = append(res, scoredEntry{Entry: e, Score: e.Base + score})
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Score == res[j].Score {
			return res[i].Entry.MTime.After(res[j].Entry.MTime)
		}
		return res[i].Score > res[j].Score
	})
	return res
}

func fuzzyScore(name, query string) (float64, bool) {
	if query == "" {
		return 0, true
	}
	ni := 0
	score := 0.0
	streak := 0
	for _, q := range query {
		found := false
		for ni < len(name) {
			r, sz := utf8.DecodeRuneInString(name[ni:])
			if r == q {
				found = true
				streak++
				score += 1 + float64(streak)*0.2
				ni += sz
				break
			}
			streak = 0
			ni += sz
		}
		if !found {
			return 0, false
		}
	}
	return score, true
}

func applyEditKey(buf *string, cursor *int, key tea.KeyMsg) bool {
	switch key.String() {
	case "backspace", "ctrl+h":
		if *cursor > 0 {
			r := []rune(*buf)
			r = append(r[:*cursor-1], r[*cursor:]...)
			*buf = string(r)
			*cursor = *cursor - 1
		}
		return true
	case "ctrl+a", "home":
		*cursor = 0
		return true
	case "ctrl+e", "end":
		*cursor = utf8.RuneCountInString(*buf)
		return true
	case "ctrl+b", "left":
		if *cursor > 0 {
			*cursor--
		}
		return true
	case "ctrl+f", "right":
		if *cursor < utf8.RuneCountInString(*buf) {
			*cursor = *cursor + 1
		}
		return true
	case "ctrl+k":
		r := []rune(*buf)
		if *cursor < len(r) {
			*buf = string(r[:*cursor])
		}
		return true
	case "ctrl+w":
		newPos := wordBoundaryBackward([]rune(*buf), *cursor)
		r := []rune(*buf)
		r = append(r[:newPos], r[*cursor:]...)
		*buf = string(r)
		*cursor = newPos
		return true
	}
	if key.Type == tea.KeyRunes && len(key.Runes) > 0 {
		r := []rune(*buf)
		ch := key.Runes[0]
		if ch < 32 {
			return false
		}
		r = append(r[:*cursor], append([]rune{ch}, r[*cursor:]...)...)
		*buf = string(r)
		*cursor = *cursor + 1
		return true
	}
	return false
}

func wordBoundaryBackward(r []rune, cursor int) int {
	if cursor <= 0 {
		return 0
	}
	pos := cursor - 1
	for pos >= 0 && !isWordRune(r[pos]) {
		pos--
	}
	for pos >= 0 && isWordRune(r[pos]) {
		pos--
	}
	return pos + 1
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func sanitizeRename(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.Join(strings.Fields(s), "-")
	return s
}

func renderInput(s string, cursor int) string {
	r := []rune(s)
	cursor = clamp(cursor, 0, len(r))
	if len(r) == 0 {
		return cursorStyle.Render(" ")
	}
	if cursor == len(r) {
		return string(r) + cursorStyle.Render(" ")
	}
	return string(r[:cursor]) + cursorStyle.Render(string(r[cursor])) + string(r[cursor+1:])
}

func datePrefix(name string) bool {
	if len(name) < 11 {
		return false
	}
	prefix := name[:11]
	_, err := time.Parse("2006-01-02-", prefix)
	return err == nil
}

func renderName(name string, width int) string {
	r := []rune(name)
	if width <= 4 || len(r) <= width {
		return name
	}
	return string(r[:maxInt(0, width-1)]) + "…"
}

func sqrt(v float64) float64 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 7; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}

func plural(n int, single, multi string) string {
	if n == 1 {
		return single
	}
	return multi
}

func clamp(v, lower, upper int) int {
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func testKeyToMsg(s string) tea.KeyMsg {
	switch s {
	case "\r":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "\x1b":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "\x1b[A":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "\x1b[B":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "\x1b[C":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "\x1b[D":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "\x7f", "\b":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "\x01":
		return tea.KeyMsg{Type: tea.KeyCtrlA}
	case "\x02":
		return tea.KeyMsg{Type: tea.KeyCtrlB}
	case "\x03":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "\x04":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "\x05":
		return tea.KeyMsg{Type: tea.KeyCtrlE}
	case "\x06":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	case "\x0b":
		return tea.KeyMsg{Type: tea.KeyCtrlK}
	case "\x0e":
		return tea.KeyMsg{Type: tea.KeyCtrlN}
	case "\x10":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	case "\x12":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "\x14":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	case "\x17":
		return tea.KeyMsg{Type: tea.KeyCtrlW}
	default:
		r := []rune(s)
		if len(r) > 0 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r[0]}}
		}
		return tea.KeyMsg{}
	}
}
