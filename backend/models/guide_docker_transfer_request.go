package models

// swagger:model guideDockerTransferRequest
type GuideDockerTransferRequest struct {

	// 忽略目录检查，直接覆盖
	Force bool `json:"force,omitempty"`

	// true则删除目录后覆盖迁移,false则不复制文件，只修改目录位置
	OverwriteDir bool `json:"overwriteDir,omitempty"`

	// docker迁移路径
	Path string `json:"path,omitempty"`
}
