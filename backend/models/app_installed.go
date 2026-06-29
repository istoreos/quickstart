package models

// swagger:model appInstalled
type AppInstalled struct {

	// arch
	Arch []string `json:"arch"`

	// author
	Author string `json:"author,omitempty"`

	// depends
	Depends []string `json:"depends"`

	// description
	Description string `json:"description,omitempty"`

	// description en
	DescriptionEn string `json:"description_en,omitempty"`

	// entry
	Entry string `json:"entry,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// release
	Release int64 `json:"release,omitempty"`

	// tags
	Tags []string `json:"tags"`

	// time
	Time int64 `json:"time,omitempty"`

	// title
	Title string `json:"title,omitempty"`

	// title en
	TitleEn string `json:"title_en,omitempty"`

	// version
	Version string `json:"version,omitempty"`

	// website
	Website string `json:"website,omitempty"`
}
