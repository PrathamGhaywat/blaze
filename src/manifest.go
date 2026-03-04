package main

type Manifest struct {
	Schema int `json:"schema"`
	Name string `json:"name"`
	Version string `json:"version"`
	Description string `json:"description"`
	Homepage string `json:"homepage"`
	Author Author `json:"author"`
	Repository Repository `json:"repository"`
	License string `json:"license"`
	Targets map[string]Target `json:"targets"`
	PostInstall map[string]string `json:"postInstall,omitempty"`
	Env map[string]string `json:"env,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	BlazeVersion string `json:"blaze,omitempty"`
}

type Author struct {
	Name string `json:"name"`
	Email string `json:"email"`
}

type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Target struct {
	ArchiveType string   `json:"archive_type"`
	URL string `json:"url"`
	SHA256 string `json:"sha256"`
	Bin []string `json:"bin"`
	ExtractRoot string `json:"extract_root,omitempty"`
}
