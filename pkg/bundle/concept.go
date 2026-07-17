// Package bundle defines the core structures for an OKF (Open Knowledge Format) bundle and its constituent concepts.
package bundle

// Frontmatter represents the metadata block of an OKF concept document.
type Frontmatter struct {
	// Type specifies the category of the concept (e.g., "PostgreSQL Table", "API Endpoint").
	Type string `yaml:"type"`
	// Title is a human-readable title for the concept.
	Title string `yaml:"title,omitempty"`
	// Desc is a short description of the concept's purpose.
	Desc string `yaml:"description,omitempty"`
	// Resource defines the physical resource name this concept maps to (e.g., table name, endpoint path).
	Resource string `yaml:"resource,omitempty"`
	// Tags are labels for categorization and searching.
	Tags []string `yaml:"tags,omitempty"`
	// Timestamp is the RFC3339 formatted generation/update time.
	Timestamp string `yaml:"timestamp,omitempty"`
	// OKFVersion is the targeted OKF version, only valid in the root index.md frontmatter.
	OKFVersion string `yaml:"okf_version,omitempty"`
	// Extra captures any other custom metadata key-value pairs inline.
	Extra map[string]interface{} `yaml:",inline"`
}

// Citation represents a single supporting source reference.
type Citation struct {
	// Number is the citation index number, e.g. 1 in [1].
	Number int `yaml:"number" json:"number"`
	// Title is the human-readable text description of the citation.
	Title string `yaml:"title" json:"title"`
	// URI is the link to the resource.
	URI string `yaml:"uri" json:"uri"`
}

// Concept represents a single OKF document (concept).
type Concept struct {
	// ID is the unique identifier (relative filepath without .md suffix, e.g., "tables/users").
	ID string
	// Path is the relative path within the bundle (e.g., "tables/users.md").
	Path string
	// Frontmatter holds the parsed metadata block.
	Frontmatter Frontmatter
	// Body is the raw Markdown body excluding the frontmatter block.
	Body string
	// ParseError contains the error description if the file could not be parsed.
	ParseError string
	// Citations holds the list of parsed citations from the concept.
	Citations []Citation
}
