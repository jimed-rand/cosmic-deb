package repos

func BuiltIn() *Config {
	return &Config{
		GeneratedAt: "2026-02-23",
		EpochLatest: "",
		Repos: []Entry{
			{Name: "cosmic-app-library", URL: "https://codeberg.org/hepp3n/cosmic-applibrary", Branch: "master"},
			{Name: "cosmic-applets", URL: "https://codeberg.org/hepp3n/cosmic-applets", Branch: "master"},
			{Name: "cosmic-bg", URL: "https://codeberg.org/hepp3n/cosmic-bg", Branch: "master"},
			{Name: "cosmic-comp", URL: "https://codeberg.org/hepp3n/cosmic-comp", Branch: "master"},
			{Name: "cosmic-edit", URL: "https://codeberg.org/hepp3n/cosmic-edit", Branch: "master"},
			{Name: "cosmic-files", URL: "https://codeberg.org/hepp3n/cosmic-files", Branch: "master"},
			{Name: "cosmic-greeter", URL: "https://codeberg.org/hepp3n/cosmic-greeter", Branch: "master"},
			{Name: "cosmic-icons", URL: "https://codeberg.org/hepp3n/cosmic-icons", Branch: "master"},
			{Name: "cosmic-idle", URL: "https://codeberg.org/hepp3n/cosmic-idle", Branch: "master"},
			{Name: "cosmic-initial-setup", URL: "https://codeberg.org/hepp3n/cosmic-initial-setup", Branch: "master"},
			{Name: "cosmic-launcher", URL: "https://codeberg.org/hepp3n/cosmic-launcher", Branch: "master"},
			{Name: "cosmic-notifications", URL: "https://codeberg.org/hepp3n/cosmic-notifications", Branch: "master"},
			{Name: "cosmic-osd", URL: "https://codeberg.org/hepp3n/cosmic-osd", Branch: "master"},
			{Name: "cosmic-panel", URL: "https://codeberg.org/hepp3n/cosmic-panel", Branch: "master"},
			{Name: "cosmic-player", URL: "https://codeberg.org/hepp3n/cosmic-player", Branch: "master"},
			{Name: "cosmic-randr", URL: "https://codeberg.org/hepp3n/cosmic-randr", Branch: "master"},
			{Name: "cosmic-screenshot", URL: "https://codeberg.org/hepp3n/cosmic-screenshot", Branch: "master"},
			{Name: "cosmic-session", URL: "https://codeberg.org/hepp3n/cosmic-session", Branch: "master"},
			{Name: "cosmic-settings", URL: "https://codeberg.org/hepp3n/cosmic-settings", Branch: "master"},
			{Name: "cosmic-settings-daemon", URL: "https://codeberg.org/hepp3n/cosmic-settings-daemon", Branch: "master"},
			{Name: "cosmic-store", URL: "https://codeberg.org/hepp3n/cosmic-store", Branch: "master"},
			{Name: "cosmic-term", URL: "https://codeberg.org/hepp3n/cosmic-term", Branch: "master"},
			{Name: "cosmic-wallpapers", URL: "https://codeberg.org/hepp3n/cosmic-wallpapers", Branch: "master"},
			{Name: "cosmic-workspaces", URL: "https://codeberg.org/hepp3n/cosmic-workspaces-epoch", Branch: "master"},
			{Name: "pop-launcher", URL: "https://codeberg.org/hepp3n/pop-launcher", Branch: "master"},
			{Name: "xdg-desktop-portal-cosmic", URL: "https://codeberg.org/hepp3n/xdg-desktop-portal-cosmic", Branch: "master"},
		},
	}
}

func MaintainerFromUpstream() (string, string) {
	return "hepp3n", "hepp3n@codeberg.org"
}
