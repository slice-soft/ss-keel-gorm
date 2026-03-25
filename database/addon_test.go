package database

import (
	"testing"
	"time"

	"github.com/slice-soft/ss-keel-core/contracts"
)

// Compile-time interface assertions (duplicate here as documentation).
var (
	_ contracts.Addon        = (*DBinstance)(nil)
	_ contracts.Debuggable   = (*DBinstance)(nil)
	_ contracts.Manifestable = (*DBinstance)(nil)
)

func newTestDBinstance() *DBinstance {
	return &DBinstance{
		events: make(chan contracts.PanelEvent, 256),
	}
}

func TestAddon_ID(t *testing.T) {
	d := newTestDBinstance()
	if got := d.ID(); got != "gorm" {
		t.Errorf("ID() = %q, want %q", got, "gorm")
	}
}

func TestAddon_PanelID(t *testing.T) {
	d := newTestDBinstance()
	if got := d.PanelID(); got != "gorm" {
		t.Errorf("PanelID() = %q, want %q", got, "gorm")
	}
}

func TestAddon_PanelLabel(t *testing.T) {
	d := newTestDBinstance()
	if got := d.PanelLabel(); got != "Database (GORM)" {
		t.Errorf("PanelLabel() = %q, want %q", got, "Database (GORM)")
	}
}

func TestAddon_Manifest(t *testing.T) {
	d := newTestDBinstance()
	m := d.Manifest()

	if m.ID != "gorm" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "gorm")
	}

	if len(m.Capabilities) != 1 || m.Capabilities[0] != "database" {
		t.Errorf("Manifest().Capabilities = %v, want [database]", m.Capabilities)
	}

	wantResources := []string{"postgres", "mysql", "sqlite", "sqlserver"}
	if len(m.Resources) != len(wantResources) {
		t.Fatalf("Manifest().Resources len = %d, want %d", len(m.Resources), len(wantResources))
	}
	for i, r := range wantResources {
		if m.Resources[i] != r {
			t.Errorf("Manifest().Resources[%d] = %q, want %q", i, m.Resources[i], r)
		}
	}

	if len(m.EnvVars) != 1 {
		t.Fatalf("Manifest().EnvVars len = %d, want 1", len(m.EnvVars))
	}
	ev := m.EnvVars[0]
	if ev.Key != "DATABASE_URL" {
		t.Errorf("EnvVars[0].Key = %q, want %q", ev.Key, "DATABASE_URL")
	}
	if !ev.Required {
		t.Error("EnvVars[0].Required should be true")
	}
	if !ev.Secret {
		t.Error("EnvVars[0].Secret should be true")
	}
	if ev.Source != "gorm" {
		t.Errorf("EnvVars[0].Source = %q, want %q", ev.Source, "gorm")
	}
}

func TestAddon_PanelEvents_ReturnsReadableChannel(t *testing.T) {
	d := newTestDBinstance()
	ch := d.PanelEvents()
	if ch == nil {
		t.Fatal("PanelEvents() returned nil channel")
	}

	// Verify it is readable by sending an event and receiving it.
	d.events <- contracts.PanelEvent{AddonID: "gorm", Label: "test"}
	select {
	case e := <-ch:
		if e.AddonID != "gorm" || e.Label != "test" {
			t.Errorf("received unexpected event: %+v", e)
		}
	default:
		t.Fatal("expected to receive event from PanelEvents channel")
	}
}

func TestAddon_TryEmit_EmitsEvent(t *testing.T) {
	d := newTestDBinstance()

	want := contracts.PanelEvent{
		Timestamp: time.Now(),
		AddonID:   "gorm",
		Label:     "query",
		Detail:    map[string]any{"operation": "query"},
		Level:     "info",
	}
	d.tryEmit(want)

	select {
	case got := <-d.PanelEvents():
		if got.AddonID != want.AddonID {
			t.Errorf("AddonID = %q, want %q", got.AddonID, want.AddonID)
		}
		if got.Label != want.Label {
			t.Errorf("Label = %q, want %q", got.Label, want.Label)
		}
		if got.Level != want.Level {
			t.Errorf("Level = %q, want %q", got.Level, want.Level)
		}
	default:
		t.Fatal("expected event to be received from PanelEvents channel")
	}
}

func TestAddon_TryEmit_DoesNotBlockWhenFull(t *testing.T) {
	d := newTestDBinstance()

	// Send 300 events to a 256-buffer channel; must not block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 300; i++ {
			d.tryEmit(contracts.PanelEvent{
				AddonID: "gorm",
				Label:   "query",
				Level:   "info",
			})
		}
	}()

	select {
	case <-done:
		// success — tryEmit did not block
	case <-time.After(2 * time.Second):
		t.Fatal("tryEmit blocked when channel was full")
	}
}
