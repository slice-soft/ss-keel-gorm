package database

import "github.com/slice-soft/ss-keel-core/contracts"

// Compile-time assertions.
var (
	_ contracts.Addon        = (*DBinstance)(nil)
	_ contracts.Debuggable   = (*DBinstance)(nil)
	_ contracts.Manifestable = (*DBinstance)(nil)
)

func (d *DBinstance) ID() string         { return "gorm" }
func (d *DBinstance) PanelID() string    { return "gorm" }
func (d *DBinstance) PanelLabel() string { return "Database (GORM)" }

func (d *DBinstance) PanelEvents() <-chan contracts.PanelEvent { return d.events }

func (d *DBinstance) Manifest() contracts.AddonManifest {
	return contracts.AddonManifest{
		ID:           "gorm",
		Version:      "1.0.0",
		Capabilities: []string{"database"},
		Resources:    []string{"postgres", "mysql", "sqlite", "sqlserver"},
		EnvVars: []contracts.EnvVar{
			{
				Key:         "DATABASE_URL",
				Description: "Database connection string (DSN). Supports Postgres, MySQL, SQLite, SQL Server.",
				Required:    true,
				Secret:      true,
				Default:     "",
				Source:      "gorm",
			},
		},
	}
}

// RegisterWithPanel registers this addon with a devpanel PanelRegistry.
// Call this from your provider setup after creating the DBinstance:
//
//	if panel, ok := app.GetAddon("devpanel").(contracts.PanelRegistry); ok {
//	    db.RegisterWithPanel(panel)
//	}
func (d *DBinstance) RegisterWithPanel(r contracts.PanelRegistry) {
	r.RegisterAddon(d)
}
