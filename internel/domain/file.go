package domain

// 文件元数据
type FileMetadata struct {
	Name    string `json:"name"`     // 文件名称
	Size    int    `json:"size"`     // 文件大小 (字节)
	Type    string `json:"type"`     // 文件类型 (例如 "image/png")
	OwnerID string `json:"owner_id"` // 上传者ID (例如用户ID)
	MD5     string `json:"md5"`      // 文件内容的MD5哈希值
	Path    string `json:"path"`
}
