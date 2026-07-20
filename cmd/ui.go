package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mhmdrz-rasekh/xray-cli/core"
	"github.com/Mhmdrz-rasekh/xray-cli/parser"
	"github.com/Mhmdrz-rasekh/xray-cli/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"
)

var (
    titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ADD8")).MarginBottom(1).MarginTop(1)
    badgeStyle     = lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(lipgloss.Color("#000000"))
    connectedBadge = badgeStyle.Copy().Background(lipgloss.Color("#00FF00"))
    offlineBadge   = badgeStyle.Copy().Background(lipgloss.Color("#FF4444"))
    dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
    highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true)
    cursorCol   = lipgloss.NewStyle().Width(4)
    indexCol    = lipgloss.NewStyle().Width(6).Align(lipgloss.Right).PaddingRight(1)
    nameCol     = lipgloss.NewStyle().Width(36)
    protoCol    = lipgloss.NewStyle().Width(10)
    pingCol     = lipgloss.NewStyle().Width(18)
)

const (
	viewMain       = "main"
	viewAddSubName = "add_sub_name"
	viewAddSubUrl  = "add_sub_url"
	viewAddLocal   = "add_local"
	viewEditNode   = "edit_node"
	viewQR         = "view_qr"
	viewAskPort    = "ask_port"
)

func killXray(process *os.Process, mode string, cfgPath string) {
	if process == nil { return }
	isRoot := os.Geteuid() == 0

	if mode == "tun" && !isRoot {
		var killCmd *exec.Cmd
		if _, err := exec.LookPath("pkexec"); err == nil { killCmd = exec.Command("pkexec", "pkill", "-f", cfgPath) } else { killCmd = exec.Command("sudo", "pkill", "-f", cfgPath) }
		_ = killCmd.Run()
	} else { _ = process.Kill() }
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit { return fmt.Sprintf("%d B", b) }
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit { div *= unit; exp++ }
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func queryXrayStats() (int64, int64) {
	xrayPath, err := exec.LookPath("xray")
	if err != nil { return 0, 0 }
	
	cmd := exec.Command(xrayPath, "api", "statsquery", "-server=127.0.0.1:10085")
	out, err := cmd.Output()
	if err != nil { return 0, 0 }

	var result struct {
		Stat []struct {
			Name  string      `json:"name"`
			Value json.Number `json:"value"`
		} `json:"stat"`
	}

	if err := json.Unmarshal(out, &result); err == nil {
		var dl, ul int64
		for _, s := range result.Stat {
			if strings.HasPrefix(s.Name, "outbound>>>") {
				val, _ := s.Value.Int64()
				if strings.HasSuffix(s.Name, ">>>downlink") { dl += val }
				if strings.HasSuffix(s.Name, ">>>uplink") { ul += val }
			}
		}
		return dl, ul
	}
	return 0, 0
}

type pingResultMsg struct {
	nodeRawLink string
	ping        string
	remaining   []storage.Node
}

type appUpdateMsg struct {
	err error
}

type tickMsg time.Time // پیام برای تایمر 1 ثانیه‌ای

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func startPingCmd(queue []storage.Node) tea.Cmd {
	if len(queue) == 0 { return nil }
	return func() tea.Msg {
		node := queue[0]
		pingStr := "\033[31m-1\033[0m"

		xrayPath, err := exec.LookPath("xray")
		if err == nil && strings.HasPrefix(node.RawLink, "vless://") {
			if parsed, err := parser.ParseVless(node.RawLink); err == nil {
				if duration, err := core.MeasureRealPing(parsed, xrayPath); err == nil {
					pingStr = fmt.Sprintf("\033[32m%d ms\033[0m", duration.Milliseconds())
				}
			}
		} else if !strings.HasPrefix(node.RawLink, "vless://") { pingStr = "\033[33mSkip\033[0m" }

		return pingResultMsg{nodeRawLink: node.RawLink, ping: pingStr, remaining: queue[1:]}
	}
}

func updateAppCmd() tea.Cmd {
	return func() tea.Msg {
		execPath, err := os.Executable()
		if err != nil { return appUpdateMsg{err: err} }
		script := fmt.Sprintf(`
		export PATH=$PATH:/usr/local/go/bin
		TMPDIR=$(mktemp -d)
		cd $TMPDIR
		git clone https://github.com/Mhmdrz-rasekh/xray-cli.git .
		go build -ldflags="-s -w" -o new-cli main.go
		if [ -w "%s" ]; then rm -f "%s" && mv new-cli "%s"
		elif command -v pkexec >/dev/null 2>&1; then pkexec sh -c 'rm -f "%s" && mv new-cli "%s"'
		else sudo -S sh -c 'rm -f "%s" && mv new-cli "%s"'; fi
		`, execPath, execPath, execPath, execPath, execPath, execPath, execPath)

		cmd := exec.Command("sh", "-c", script)
		out, err := cmd.CombinedOutput()
		if err != nil { return appUpdateMsg{err: fmt.Errorf("%v: %s", err, string(out))} }
		return appUpdateMsg{err: nil}
	}
}

var (
    reData = regexp.MustCompile(`(\d+|نامحدود)\s*گیگابایت`)
    reDays = regexp.MustCompile(`(\d+|نامحدود)\s*روز`)
    rePing = regexp.MustCompile(`(\d+)\s*ms`)
)

func fetchSubscription(urlStr, groupName string) ([]storage.Node, map[string]string, error) {
    resp, err := http.Get(urlStr)
    if err != nil { return nil, nil, err }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil { return nil, nil, err }

    content := strings.TrimSpace(string(body))
    if pad := len(content) % 4; pad != 0 { content += strings.Repeat("=", 4-pad) }
    decoded, err := base64.StdEncoding.DecodeString(content)
    if err != nil { decoded, err = base64.URLEncoding.DecodeString(content) }
    if err == nil { content = string(decoded) }

    var newNodes []storage.Node
    metadata := make(map[string]string)

    lines := strings.Split(content, "\n")
	for _, line := range lines {
        line = strings.TrimSpace(line)
        if !strings.HasPrefix(line, "vless://") { continue }

        parsed, err := parser.ParseVless(line)
        name := ""
        if err == nil { name = parsed.Name }

        isDummy := reData.MatchString(name) || reDays.MatchString(name) || 
                   strings.Contains(name, "اسم") || strings.Contains(name, "ترافیک") || 
                   strings.Contains(name, "زمان") || strings.Contains(name, "انقضا") ||
                   strings.Contains(name, "حجم")

        if isDummy {
            // Data Metadata Extraction
            if m := reData.FindStringSubmatch(name); m != nil {
                if m[1] == "نامحدود" { metadata["usage"] = "∞" } else { metadata["usage"] = m[1] + " GB" }
            } else if (strings.Contains(name, "ترافیک") || strings.Contains(name, "حجم")) && strings.Contains(name, "نامحدود") {
                metadata["usage"] = "∞"
            }

            // Days Metadata Extraction
            if m := reDays.FindStringSubmatch(name); m != nil {
                if m[1] == "نامحدود" { metadata["days"] = "∞" } else { metadata["days"] = m[1] + " Days" }
            } else if (strings.Contains(name, "زمان") || strings.Contains(name, "انقضا")) && strings.Contains(name, "نامحدود") {
                metadata["days"] = "∞"
            }
            continue // Drop dummy node
        }
        
        if parsed != nil { name = parsed.Name } else { name = "Sub Node" }
        newNodes = append(newNodes, storage.Node{Name: name, Protocol: "VLESS", RawLink: line, Group: groupName})
    }
	return newNodes, metadata, nil
}

func getPingValue(pingStr string) int {
    if match := rePing.FindStringSubmatch(pingStr); match != nil {
        if val, err := strconv.Atoi(match[1]); err == nil { return val }
    }
    return -1
}

func sortNodes(nodes []storage.Node, mode string) []storage.Node {
    sort.SliceStable(nodes, func(i, j int) bool {
        // 1. Preserve Group Headers
        if nodes[i].Group == "Local" && nodes[j].Group != "Local" { return true }
        if nodes[i].Group != "Local" && nodes[j].Group == "Local" { return false }
        if nodes[i].Group != nodes[j].Group { return nodes[i].Group < nodes[j].Group }

        // 2. Sort within the group
        if mode == "ping" {
            p1, p2 := getPingValue(nodes[i].Ping), getPingValue(nodes[j].Ping)
            if p1 != p2 {
                if p1 == -1 { return false } // Dead node sinks
                if p2 == -1 { return true }  // Dead node sinks
                return p1 < p2
            }
        }

        // 3. Fallback: Sort by Name Alphabetically
        name1, name2 := nodes[i].Name, nodes[j].Name
        if p1, err := parser.ParseVless(nodes[i].RawLink); err == nil { name1 = p1.Name }
        if p2, err := parser.ParseVless(nodes[j].RawLink); err == nil { name2 = p2.Name }
        return strings.ToLower(name1) < strings.ToLower(name2)
    })
    return nodes
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive terminal dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.LoadDB()
		if err != nil {
			fmt.Printf("Error loading database: %v\n", err)
			return
		}
		db.Nodes = sortNodes(db.Nodes, "name")

		ti := textinput.New()
		ti.Focus()
		ti.Width = 60
		var editInputs []textinput.Model
		for i := 0; i < 2; i++ {
			t := textinput.New()
			t.Width = 80
			editInputs = append(editInputs, t)
		}

		m := model{
            nodes: db.Nodes, currentView: viewMain, textInput: ti, dbRef: db,
            editInputs: editInputs, editFocus: 0,
            subMetadata: make(map[string]string),
			sortMode: "name",
        }

		p := tea.NewProgram(m, tea.WithAltScreen())
		runModel, _ := p.Run()

		if runModel != nil {
			finalModel := runModel.(model)
			if finalModel.isConnected {
				killXray(finalModel.xrayProcess, finalModel.connectedMode, finalModel.connectedCfgPath)
				if finalModel.connectedMode == "system" { core.DisableSystemProxy() }
			}
		}
	},
}

type model struct {
	nodes       []storage.Node
	cursor      int
	currentView string
	tempSubName string
	textInput   textinput.Model
	statusMsg   string
	dbRef       *storage.DB
	editInputs  []textinput.Model
	editFocus   int
	qrString    string

	subMetadata map[string]string
	sortMode    string
	
	terminalHeight int
	terminalWidth  int
	viewportStart  int

	pendingMode      string
	isConnected      bool
	connectedNode    *storage.Node
	connectedMode    string
	connectedCfgPath string
	xrayProcess      *os.Process

	// متغیرهای ذخیره اطلاعات شبکه
	dlTotal int64
	ulTotal int64
	dlSpeed int64
	ulSpeed int64

	showHelp         bool
}

// در لحظه استارت شدن برنامه، تایمر هم فعال می‌شود
func (m model) Init() tea.Cmd { return tea.Batch(textinput.Blink, doTick()) }

func (m model) getVisibleLimit() int {
	maxVis := 10
	if m.terminalHeight > 24 { maxVis = m.terminalHeight - 22 }
	if maxVis < 4 { maxVis = 4 }
	return maxVis
}

func (m model) ensureViewport() model {
	if m.terminalHeight == 0 || len(m.nodes) == 0 { return m }
	availableLines := m.terminalHeight - 22
	if availableLines < 5 { availableLines = 5 }

	for {
		if m.cursor < m.viewportStart { m.viewportStart = m.cursor; break }
		linesUsed := 0
		end := m.viewportStart
		var lastGrp string
		if m.viewportStart > 0 { lastGrp = m.nodes[m.viewportStart-1].Group }
		for i := m.viewportStart; i < len(m.nodes); i++ {
			needed := 1
			if m.nodes[i].Group != lastGrp { needed += 2; lastGrp = m.nodes[i].Group }
			if linesUsed+needed > availableLines { break }
			linesUsed += needed; end = i + 1
		}
		if m.cursor >= end { m.viewportStart++ } else { break }
	}
	return m
}

func (m model) startConnection(mode string, port int) model {
	if m.isConnected {
		killXray(m.xrayProcess, m.connectedMode, m.connectedCfgPath)
		if m.connectedMode == "system" { core.DisableSystemProxy() }
		m.dlTotal, m.ulTotal, m.dlSpeed, m.ulSpeed = 0, 0, 0, 0           // reset traffic on new connection
	}

	xrayPath, err := exec.LookPath("xray")
	if err != nil { m.statusMsg = "Error: 'xray' binary not found in PATH"; return m }

	node := m.nodes[m.cursor]
	parsed, err := parser.ParseVless(node.RawLink)
	if err == nil {
		cfgPath, err := core.GenerateConfig(parsed, mode, port)
		if err == nil {
			if mode == "system" { core.EnableSystemProxy() }
			var xrayCmd *exec.Cmd
			isRoot := os.Geteuid() == 0

			if mode == "tun" && !isRoot {
				m.statusMsg = "Requesting Root for TUN..."
				if _, err := exec.LookPath("pkexec"); err == nil { xrayCmd = exec.Command("pkexec", xrayPath, "run", "-c", cfgPath) } else { xrayCmd = exec.Command("sudo", xrayPath, "run", "-c", cfgPath) }
			} else { xrayCmd = exec.Command(xrayPath, "run", "-c", cfgPath) }

			logPath := filepath.Join(os.TempDir(), "xray-cli.log")
			logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			xrayCmd.Stdout, xrayCmd.Stderr = logFile, logFile

			if err := xrayCmd.Start(); err != nil {
				if mode == "system" { core.DisableSystemProxy() }
				m.statusMsg = "Failed to start: Authentication canceled."
			} else {
				go func() { _ = xrayCmd.Wait() }()
				m.isConnected, m.connectedMode, m.connectedNode, m.connectedCfgPath, m.xrayProcess = true, mode, &node, cfgPath, xrayCmd.Process
				m.statusMsg = fmt.Sprintf("Started on port %d! Logs saved to %s", port, logPath)
			}
		} else { m.statusMsg = "Config err: " + err.Error() }
	}
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.currentView == viewAddSubName || m.currentView == viewAddSubUrl || m.currentView == viewAddLocal || m.currentView == viewAskPort { m.textInput, cmd = m.textInput.Update(msg)
	} else if m.currentView == viewEditNode { m.editInputs[m.editFocus], cmd = m.editInputs[m.editFocus].Update(msg) }

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width; m.terminalHeight = msg.Height
		return m.ensureViewport(), nil

	case tickMsg:
		if m.isConnected {
			dl, ul := queryXrayStats()
			if m.dlTotal > 0 || dl > 0 { 
				m.dlSpeed = dl - m.dlTotal
				if m.dlSpeed < 0 { m.dlSpeed = dl } 
				m.ulSpeed = ul - m.ulTotal
				if m.ulSpeed < 0 { m.ulSpeed = ul }
			}
			m.dlTotal = dl
			m.ulTotal = ul
		} else {
			m.dlTotal, m.ulTotal, m.dlSpeed, m.ulSpeed = 0, 0, 0, 0
		}
		return m, doTick()

	case appUpdateMsg:
		if msg.err != nil { m.statusMsg = "Update failed: " + msg.err.Error() } else { m.statusMsg = "Update installed! Restart the app to apply changes." }
		return m.ensureViewport(), nil

	case pingResultMsg:
		for i, n := range m.nodes { if n.RawLink == msg.nodeRawLink { m.nodes[i].Ping = msg.ping; break } }
		if len(msg.remaining) > 0 { return m, startPingCmd(msg.remaining) }
		m.statusMsg = "Ping tasks completed."
		return m.ensureViewport(), nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c": return m, tea.Quit
		case "q", "esc":
			if m.currentView != viewMain { m.currentView = viewMain; m.textInput.Reset(); m.statusMsg = ""; return m.ensureViewport(), nil }
			return m, tea.Quit
		case "ctrl+u":
			if m.currentView == viewMain { m.statusMsg = "Downloading & installing update from GitHub... Please wait."; return m, updateAppCmd() }
		case "tab", "shift+tab":
			if m.currentView == viewEditNode {
				if msg.String() == "shift+tab" { m.editFocus-- } else { m.editFocus++ }
				if m.editFocus > 1 { m.editFocus = 0 }
				if m.editFocus < 0 { m.editFocus = 1 }
				for i := range m.editInputs { if i == m.editFocus { m.editInputs[i].Focus() } else { m.editInputs[i].Blur() } }
				return m, nil
			}
		case "up", "down", "k", "j":
            if m.currentView == viewEditNode {
                if msg.String() == "up" || msg.String() == "k" { m.editFocus-- } else { m.editFocus++ }
                if m.editFocus > 1 { m.editFocus = 0 }
                if m.editFocus < 0 { m.editFocus = 1 }
                for i := range m.editInputs {
                    if i == m.editFocus { m.editInputs[i].Focus() } else { m.editInputs[i].Blur() }
                }
                return m, nil
            }

            if m.currentView == viewMain {
                if (msg.String() == "up" || msg.String() == "k") && m.cursor > 0 { m.cursor-- }
                if (msg.String() == "down" || msg.String() == "j") && m.cursor < len(m.nodes)-1 { m.cursor++ }
                return m.ensureViewport(), nil
            }
		case "x":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				m.nodes = append(m.nodes[:m.cursor], m.nodes[m.cursor+1:]...); m.dbRef.Nodes = m.nodes; _ = storage.SaveDB(m.dbRef); m.statusMsg = "Config deleted."
				if m.cursor >= len(m.nodes) && m.cursor > 0 { m.cursor-- }
				return m.ensureViewport(), nil
			}
		case "X":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				groupToDelete := m.nodes[m.cursor].Group
				if groupToDelete == "Local" { m.statusMsg = "Cannot delete 'Local' group at once. Delete nodes with 'x'."; return m, nil }
				var filteredNodes []storage.Node
				for _, n := range m.nodes { if n.Group != groupToDelete { filteredNodes = append(filteredNodes, n) } }
				m.nodes = filteredNodes
				var filteredSubs []storage.Subscription
				for _, s := range m.dbRef.Subscriptions { if s.Name != groupToDelete { filteredSubs = append(filteredSubs, s) } }
				m.dbRef.Subscriptions = filteredSubs; m.dbRef.Nodes = m.nodes; _ = storage.SaveDB(m.dbRef); m.statusMsg = fmt.Sprintf("Subscription [%s] completely deleted!", groupToDelete); m.cursor = 0; m.viewportStart = 0; return m.ensureViewport(), nil
			}
		case "u", "U":
            if m.currentView == viewMain {
                m.statusMsg = "Updating subscriptions..."; var newNodes []storage.Node
                
                // LEAVE THIS LOOP ALONE (Preserves Local configs)
                for _, n := range m.nodes { if n.Group == "Local" { newNodes = append(newNodes, n) } }
                
                // REPLACE YOUR SECOND LOOP WITH THIS MULTI-LINE BLOCK
                for _, sub := range m.dbRef.Subscriptions { 
                    fetched, meta, _ := fetchSubscription(sub.URL, sub.Name)
                    if len(meta) > 0 { 
                        m.subMetadata[sub.Name] = fmt.Sprintf("%s | %s", meta["usage"], meta["days"]) 
                    }
                    newNodes = append(newNodes, fetched...) 
                }
                
                m.nodes = sortNodes(newNodes, m.sortMode); m.dbRef.Nodes = m.nodes; _ = storage.SaveDB(m.dbRef); m.statusMsg = "All subscriptions updated!"; if m.cursor >= len(m.nodes) { m.cursor = 0; m.viewportStart = 0 }; return m.ensureViewport(), nil
            }
		case "o", "O":
            if m.currentView == viewMain && len(m.nodes) > 0 {
                if m.sortMode == "ping" { 
                    m.sortMode = "name" 
                } else { 
                    m.sortMode = "ping" 
                }
                m.nodes = sortNodes(m.nodes, m.sortMode)
                m.dbRef.Nodes = m.nodes
                _ = storage.SaveDB(m.dbRef)
                m.statusMsg = "Order: Sorted by " + strings.ToUpper(m.sortMode)
                return m.ensureViewport(), nil
            }
		case "v", "V":
			if m.currentView == viewMain && len(m.nodes) > 0 { m.currentView = viewQR; var buf strings.Builder; qrterminal.GenerateHalfBlock(m.nodes[m.cursor].RawLink, qrterminal.L, &buf); m.qrString = buf.String(); return m, nil }
		case "e", "E":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				node := m.nodes[m.cursor]; m.currentView = viewEditNode; m.editFocus = 0; name := node.Name
				if parsed, err := parser.ParseVless(node.RawLink); err == nil { name = parsed.Name }
				m.editInputs[0].SetValue(name); m.editInputs[0].Focus(); m.editInputs[1].SetValue(node.RawLink); m.editInputs[1].Blur(); return m, nil
			}
		case "p", "P":
			if m.currentView == viewMain && len(m.nodes) > 0 { m.nodes[m.cursor].Ping = "..."; m.statusMsg = "Pinging selected node..."; return m.ensureViewport(), startPingCmd([]storage.Node{m.nodes[m.cursor]}) }
		case "g", "G":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				group := m.nodes[m.cursor].Group; var queue []storage.Node
				for i, n := range m.nodes { if n.Group == group { m.nodes[i].Ping = "..."; queue = append(queue, n) } }
				m.statusMsg = fmt.Sprintf("Pinging group: %s...", group); return m.ensureViewport(), startPingCmd(queue)
			}
		case "c", "C":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				var queue []storage.Node
				for i, n := range m.nodes { m.nodes[i].Ping = "..."; queue = append(queue, n) }
				m.statusMsg = "Checking all nodes sequentially..."; return m.ensureViewport(), startPingCmd(queue)
			}
		case "a", "A":
			if m.currentView == viewMain { m.currentView = viewAddSubName; m.textInput.Placeholder = "Enter Name (e.g. Sub1)"; m.textInput.Focus(); return m, nil }
		case "l", "L":
			if m.currentView == viewMain { m.currentView = viewAddLocal; m.textInput.Placeholder = "Paste raw vless:// link"; m.textInput.Focus(); return m, nil }
		case "enter":
			switch m.currentView {
			case viewAddSubName:
				name := strings.TrimSpace(m.textInput.Value()); if name != "" { m.tempSubName = name; m.currentView = viewAddSubUrl; m.textInput.Reset(); m.textInput.Placeholder = "Paste URL" }
		case viewAddSubUrl:
                urlStr := strings.TrimSpace(m.textInput.Value())
                if urlStr != "" {
                    m.dbRef.Subscriptions = append(m.dbRef.Subscriptions, storage.Subscription{Name: m.tempSubName, URL: urlStr})
                    
                    // --- REPLACE THESE SPECIFIC LINES ---
                    fetched, meta, err := fetchSubscription(urlStr, m.tempSubName)
                    if err == nil { 
                        if len(meta) > 0 { 
                            m.subMetadata[m.tempSubName] = fmt.Sprintf("%s | %s", meta["usage"], meta["days"]) 
                        }
                        m.nodes = append(m.nodes, fetched...)
                        m.statusMsg = fmt.Sprintf("Added %s!", m.tempSubName) 
                    }
                    // ------------------------------------
                    
                    m.nodes = sortNodes(m.nodes, m.sortMode); m.dbRef.Nodes = m.nodes; _ = storage.SaveDB(m.dbRef); m.currentView = viewMain; m.textInput.Reset()
                }
			case viewEditNode:
				newName := strings.TrimSpace(m.editInputs[0].Value()); newLink := strings.TrimSpace(m.editInputs[1].Value())
				if newName != "" && strings.HasPrefix(newLink, "vless://") {
					parts := strings.Split(newLink, "#"); finalLink := parts[0] + "#" + newName
					m.nodes[m.cursor].Name = newName; m.nodes[m.cursor].RawLink = finalLink; m.dbRef.Nodes = m.nodes; _ = storage.SaveDB(m.dbRef); m.statusMsg = "Config updated successfully."; m.currentView = viewMain
				} else { m.statusMsg = "Error: Invalid VLESS link format." }
			case viewAskPort:
				portStr := strings.TrimSpace(m.textInput.Value()); port := 10808
				if portStr != "" { if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 { port = p } else { m.statusMsg = "Invalid port! Using default 10808." } }
				m.currentView = viewMain; m.textInput.Reset(); return m.startConnection(m.pendingMode, port).ensureViewport(), nil
			}
			return m.ensureViewport(), nil
		case "m", "M", "s", "S", "t", "T":
			if m.currentView == viewMain && len(m.nodes) > 0 {
				mode := "manual"
				if msg.String() == "s" || msg.String() == "S" { mode = "system" }
				if msg.String() == "t" || msg.String() == "T" { mode = "tun" }
				if mode == "manual" { m.pendingMode = "manual"; m.currentView = viewAskPort; m.textInput.Placeholder = "Enter SOCKS port (Press Enter for 10808)"; m.textInput.Focus(); return m, nil }
				return m.startConnection(mode, 10808).ensureViewport(), nil
			}
		case "?":
            if m.currentView == viewMain {
                m.showHelp = !m.showHelp
                return m.ensureViewport(), nil
            }
		case "d", "D":
			if m.currentView == viewMain && m.isConnected {
				killXray(m.xrayProcess, m.connectedMode, m.connectedCfgPath)
				if m.connectedMode == "system" { core.DisableSystemProxy() }
				m.isConnected, m.connectedNode, m.connectedMode, m.connectedCfgPath, m.xrayProcess = false, nil, "", "", nil
				m.dlTotal, m.ulTotal, m.dlSpeed, m.ulSpeed = 0, 0, 0, 0
				m.statusMsg = "Disconnected successfully."
				return m.ensureViewport(), nil
			}
		}
	}
	return m, cmd
}

func (m model) View() string {
	var s strings.Builder
	// 1. Render the Main Title
    s.WriteString(titleStyle.Render("XRAY-CLI DASHBOARD") + "\n")

    // 2. Render the Status Line
    if m.currentView == viewMain || m.currentView == viewEditNode {
        if m.isConnected && m.connectedNode != nil {
            name := m.connectedNode.Name
            if parsed, err := parser.ParseVless(m.connectedNode.RawLink); err == nil { name = parsed.Name }
            
            status := connectedBadge.Render("CONNECTED")
            mode := highlightStyle.Render(strings.ToUpper(m.connectedMode))
            server := highlightStyle.Render(name)
            
            s.WriteString(fmt.Sprintf("%s  Mode: %s  |  Server: %s\n", status, mode, server))
		    // چاپ سرعت و مصرف ترافیک در خط دوم به صورت زنده
		    statsLine := fmt.Sprintf("   \033[36m▼ %s/s\033[0m (%s)   \033[35m▲ %s/s\033[0m (%s)", 
				    formatBytes(m.dlSpeed), formatBytes(m.dlTotal),
				    formatBytes(m.ulSpeed), formatBytes(m.ulTotal))
		    s.WriteString(statsLine + "\n")
        } else {
            s.WriteString(fmt.Sprintf("%s\n", offlineBadge.Render("DISCONNECTED")))
        }
        s.WriteString(dimStyle.Render(strings.Repeat("━", 86)) + "\n")
    }

 // 3. Render the Conditional Help Menu
    if m.currentView == viewMain {
        if m.showHelp {
            // Column 1: Navigation & System
            c1 := lipgloss.JoinVertical(lipgloss.Left,
                "[↑/↓ | j/k] Navigate",
				"[O] Order: Name/Ping",
                "[?] Hide Help",
                "[Q] Quit",
                "[Ctrl+U] Update App",
                "",
            )
            // Column 2: Connection Modes
            c2 := lipgloss.JoinVertical(lipgloss.Left,
                "[M] Manual",
                "[S] SysProxy",
                "[T] TUN Mode",
                "[D] Disconnect",
                "",
            )
            // Column 3: Node Management
            c3 := lipgloss.JoinVertical(lipgloss.Left,
                "[L] Add Local",
                "[A] Add Sub",
                "[E] Edit Node",
                "[x] Del Node",
                "[Shift+X] Del Sub",
            )
            // Column 4: Actions & Utilities
            c4 := lipgloss.JoinVertical(lipgloss.Left,
                "[P] Ping Node",
                "[G] Ping Grp",
                "[C] Ping All",
                "[U] Update Subs",
                "[V] View QR Code",
            )

            // Stitch columns together with rigid widths
            helpGrid := lipgloss.JoinHorizontal(lipgloss.Top,
                lipgloss.NewStyle().Width(24).Render(c1),
                lipgloss.NewStyle().Width(18).Render(c2),
                lipgloss.NewStyle().Width(22).Render(c3),
                lipgloss.NewStyle().Width(20).Render(c4),
            )
            
            s.WriteString(dimStyle.Render(helpGrid) + "\n")
        } else {
            s.WriteString(dimStyle.Render("Press [?] for help • [Q] Quit") + "\n")
        }
        s.WriteString(dimStyle.Render(strings.Repeat("━", 86)) + "\n\n")
    }


	switch m.currentView {
    case viewMain:
		if len(m.nodes) == 0 {
			s.WriteString("\n[ ▼ LOCAL ]\n( No configs. Press 'A' or 'L' to add one. )\n\n")
		} else {
			availableLines := m.terminalHeight - 22
			if availableLines < 5 { availableLines = 5 }

			linesUsed := 0; end := m.viewportStart; var tempLastGrp string
			if m.viewportStart > 0 { tempLastGrp = m.nodes[m.viewportStart-1].Group }

			for i := m.viewportStart; i < len(m.nodes); i++ {
				needed := 1; if m.nodes[i].Group != tempLastGrp { needed += 2; tempLastGrp = m.nodes[i].Group }
				if linesUsed+needed > availableLines { break }
				linesUsed += needed; end = i + 1
			}

			if m.viewportStart > 0 { s.WriteString("... ⇡ (Scroll Up) ⇡ ...\n") } else { s.WriteString("\n") }
			var renderLastGrp string; if m.viewportStart > 0 { renderLastGrp = m.nodes[m.viewportStart-1].Group }

			for i := m.viewportStart; i < end; i++ {
				node := m.nodes[i]
				
				// 1. Group Header
				if node.Group != renderLastGrp {
					metaStr := ""
					if meta, exists := m.subMetadata[node.Group]; exists && meta != " | " && meta != "" {
						metaStr = fmt.Sprintf(" ( %s )", meta)
					}
					s.WriteString(fmt.Sprintf("\n[ ▼ %s%s ]\n", strings.ToUpper(node.Group), metaStr))
					renderLastGrp = node.Group
				}

				if node.Group != renderLastGrp {
					s.WriteString(fmt.Sprintf("\n[ ▼ %s ]\n", strings.ToUpper(node.Group)))
					renderLastGrp = node.Group
				}
				
				// 2. Cursor and State Markers
				marker := "   "
				if m.cursor == i { marker = "▶  " }
				if m.isConnected && m.connectedNode != nil && m.connectedNode.RawLink == node.RawLink {
					if m.cursor == i {
						marker = "▶★ "
					} else {
						marker = " ★ "
					}
				}

				// 3. Name Sanitization (No manual space padding needed)
				displayName := node.Name
				if parsed, err := parser.ParseVless(node.RawLink); err == nil { displayName = parsed.Name }
				
				runes := []rune(displayName)
				if len(runes) > 34 { 
					displayName = string(runes[:33]) + "…" 
            }

				// 4. Ping Formatting
				pingStr := ""
				if node.Ping == "..." { 
					pingStr = "⏳" 
				} else if node.Ping != "" { 
					pingStr = fmt.Sprintf("⟪ %s ⟫", node.Ping) 
				}

				// 5. Construct the Row using Lipgloss Columns
				row := lipgloss.JoinHorizontal(lipgloss.Left,
					cursorCol.Render(marker),
					indexCol.Render(fmt.Sprintf("[%d]", i+1)),
					nameCol.Render(displayName),
					protoCol.Render(fmt.Sprintf("(%s)", node.Protocol)),
					pingCol.Render(pingStr),
				)
				
				s.WriteString(row + "\n")
        }
			if end < len(m.nodes) { s.WriteString("\n... ⇣ (Scroll Down) ⇣ ...\n") } else { s.WriteString("\n\n") }
		}

	case viewQR:
		s.WriteString("=== QR Code ===\nPress [ESC] or [Q] to return\n" + strings.Repeat("━", 86) + "\n\n" + m.qrString + "\n")
	case viewEditNode:
		s.WriteString("=== Edit Configuration ===\n[Tab] Switch Fields | [Enter] Save | [ESC] Cancel\n" + strings.Repeat("━", 86) + "\n\nAlias / Name:\n" + m.editInputs[0].View() + "\n\nRaw Link (vless://...):\n" + m.editInputs[1].View() + "\n")
	case viewAskPort:
		s.WriteString("=== Start Manual Connection ===\nLeave empty to use default SOCKS port (10808).\n" + strings.Repeat("━", 86) + "\n\n" + m.textInput.View() + "\n")
	case viewAddSubName: s.WriteString("Step 1/2: Sub Name\n\n" + m.textInput.View() + "\n")
	case viewAddSubUrl: s.WriteString(fmt.Sprintf("Step 2/2: URL for [%s]\n\n", m.tempSubName) + m.textInput.View() + "\n")
	case viewAddLocal: s.WriteString("Add Raw Config (Local)\n\n" + m.textInput.View() + "\n")
	}

	s.WriteString("\n" + strings.Repeat("━", 86) + "\n")
	if m.statusMsg != "" { s.WriteString("📢 Status: " + m.statusMsg + "\n") } else { s.WriteString("\n") }
	return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Center, s.String())
}

func init() { rootCmd.AddCommand(uiCmd) }
