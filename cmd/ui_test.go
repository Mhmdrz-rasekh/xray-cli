package cmd

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mhmdrz-rasekh/xray-cli/storage"
	tea "github.com/charmbracelet/bubbletea"
)

const testLinkA = "vless://11111111-1111-1111-1111-111111111111@a.example.com:443?security=tls&type=tcp#Server-A"
const testLinkB = "vless://22222222-2222-2222-2222-222222222222@b.example.com:443?security=tls&type=tcp#Server-B"

func newTestModel(t *testing.T, subs []storage.Subscription, nodes []storage.Node) model {
	t.Helper()
	// Keep SaveDB away from the real user config.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	db := &storage.DB{Subscriptions: subs, Nodes: nodes}
	return model{
		nodes:       nodes,
		currentView: viewMain,
		dbRef:       db,
		subMetadata: make(map[string]string),
		sortMode:    "name",
	}
}

func subServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(body))))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestUpdateFlagRegistered(t *testing.T) {
	f := uiCmd.Flags().Lookup("update")
	if f == nil {
		t.Fatal("--update flag is not registered on the ui command")
	}
	if f.Shorthand != "u" {
		t.Fatalf("expected shorthand -u, got %q", f.Shorthand)
	}
	if f.DefValue != "false" {
		t.Fatalf("flag must default to false (fast boot), got %q", f.DefValue)
	}
}

func TestInitWithoutFlagDoesNotUpdate(t *testing.T) {
	m := newTestModel(t, nil, nil)
	if m.autoUpdate {
		t.Fatal("autoUpdate must be off by default")
	}
	// Run the batch and make sure no startupUpdateMsg is produced.
	msg := m.Init()()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil { continue }
			if _, isStartup := c().(startupUpdateMsg); isStartup {
				t.Fatal("Init triggered a startup update without the flag")
			}
		}
	}
}

func TestInitWithFlagEmitsStartupUpdate(t *testing.T) {
	m := newTestModel(t, nil, nil)
	m.autoUpdate = true
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok {
		t.Fatal("expected Init to return a batch")
	}
	found := false
	for _, c := range batch {
		if c == nil { continue }
		// textinput.Blink / doTick may block or return other msgs; only call
		// commands that are instantaneous by checking via goroutine result type.
		if _, isStartup := c().(startupUpdateMsg); isStartup { found = true }
	}
	if !found {
		t.Fatal("Init with autoUpdate did not emit startupUpdateMsg")
	}
}

func TestStartupUpdateMsgRefreshesSubscriptions(t *testing.T) {
	srv := subServer(t, testLinkA+"\n"+testLinkB+"\n")
	local := storage.Node{Name: "mine", Protocol: "VLESS", RawLink: testLinkA, Group: "Local"}
	stale := storage.Node{Name: "old", Protocol: "VLESS", RawLink: "vless://x@old.example.com:1#old", Group: "MySub"}
	m := newTestModel(t, []storage.Subscription{{Name: "MySub", URL: srv.URL}}, []storage.Node{local, stale})
	m.autoUpdate = true

	out, _ := m.Update(startupUpdateMsg{})
	got := out.(model)

	if got.statusMsg != "All subscriptions updated!" {
		t.Fatalf("unexpected status: %q", got.statusMsg)
	}
	var sub, loc int
	for _, n := range got.nodes {
		switch n.Group {
		case "MySub":
			sub++
			if n.RawLink == stale.RawLink { t.Fatal("stale node survived a successful update") }
		case "Local":
			loc++
		}
	}
	if sub != 2 { t.Fatalf("expected 2 subscription nodes, got %d", sub) }
	if loc != 1 { t.Fatalf("local node must be preserved, got %d", loc) }

	saved, err := storage.LoadDB()
	if err != nil { t.Fatal(err) }
	if len(saved.Nodes) != 3 { t.Fatalf("expected 3 persisted nodes, got %d", len(saved.Nodes)) }
}

func TestManualUpdateKeyUsesSameLogic(t *testing.T) {
	srv := subServer(t, testLinkA+"\n")
	m := newTestModel(t, []storage.Subscription{{Name: "S", URL: srv.URL}}, nil)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	got := out.(model)
	if len(got.nodes) != 1 || got.statusMsg != "All subscriptions updated!" {
		t.Fatalf("manual 'u' failed: nodes=%d status=%q", len(got.nodes), got.statusMsg)
	}
}

func TestOfflineUpdateKeepsExistingNodes(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // now unreachable
	old := storage.Node{Name: "Server-A", Protocol: "VLESS", RawLink: testLinkA, Group: "S"}
	m := newTestModel(t, []storage.Subscription{{Name: "S", URL: url}}, []storage.Node{old})

	out, _ := m.Update(startupUpdateMsg{})
	got := out.(model)
	if len(got.nodes) != 1 || got.nodes[0].RawLink != testLinkA {
		t.Fatalf("offline update must not wipe saved nodes, got %+v", got.nodes)
	}
	if got.statusMsg == "All subscriptions updated!" {
		t.Fatal("status should report the failure")
	}
}

func TestStartupUpdateIgnoredOutsideMainView(t *testing.T) {
	srv := subServer(t, testLinkA+"\n")
	m := newTestModel(t, []storage.Subscription{{Name: "S", URL: srv.URL}}, nil)
	m.currentView = viewAddSubName
	out, _ := m.Update(startupUpdateMsg{})
	if len(out.(model).nodes) != 0 { t.Fatal("update must not run outside the main view") }
}
