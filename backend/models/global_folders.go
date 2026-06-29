package models

// swagger:model globalFolders
type GlobalFolders struct {

	// Caches文件夹，默认 $Home/Caches
	// Example: /mnt/sata1-1/Caches
	Caches string `json:"caches,omitempty"`

	// Configs文件夹，默认 $Home/Configs
	// Example: /mnt/sata1-1/Configs
	Configs string `json:"configs,omitempty"`

	// Downloads文件夹，默认 $Public/Downloads
	// Example: /mnt/sata1-1/Public/Downloads
	Downloads string `json:"downloads,omitempty"`

	// Home文件夹
	// Example: /mnt/sata1-1
	Home string `json:"home,omitempty"`

	// Public文件夹，默认 $Home/Public
	// Example: /mnt/sata1-1/Public
	Public string `json:"public,omitempty"`
}
