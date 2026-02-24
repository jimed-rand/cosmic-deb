package repos

type Entry struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Tag    string `json:"tag"`
	Branch string `json:"branch,omitempty"`
}

type Config struct {
	GeneratedAt string  `json:"generated_at"`
	EpochLatest string  `json:"epoch_latest"`
	Repos       []Entry `json:"repos"`
}
