package ufs

import (
	"fmt"
	"gorm.io/gorm"
	"path/filepath"
)

// Persistor is the interface for persistence operations.
type Persistor interface {
	PersistFile(path string, isDir bool) error
	RemovePersistedFile(path string) error
	LoadDirMap(path string) (dirMap map[string][]string, err error)
	UpdatePaths(srcPath, dstPath string) error
	PathExists(path string) bool
}

type FileSystem struct {
	ID          uint        `gorm:"column:id;primaryKey;autoIncrement"`                                // 自动递增主键，列名为 "id"
	Name        string      `gorm:"column:name;size:255;not null"`                                     // 文件或目录名称，最大长度255，非空，列名为 "name"
	Path        string      `gorm:"column:path;size:1024;not null;uniqueIndex"`                        // 文件或目录路径，最大长度1024，非空，唯一索引，列名为 "path"
	ParentID    uint        `gorm:"column:parent_id;index"`                                            // 父目录ID，普通索引，列名为 "parent_id"
	IsDirectory bool        `gorm:"column:is_directory;not null;default:false"`                        // 是否为目录，默认为false（文件），列名为 "is_directory"
	Content     []byte      `gorm:"column:content;type:blob"`                                          // 文件内容，BLOB类型，列名为 "content"
	Parent      *FileSystem `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 外键，父目录ID，级联更新和删除
}

type GormPersistor struct {
	db *gorm.DB
}

func NewGormPersistor(db *gorm.DB) *GormPersistor {
	return &GormPersistor{db: db}
}

func (p *GormPersistor) PersistFile(path string, isDir bool) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		absPath := filepath.Clean(path)
		dirPath := filepath.Dir(absPath)

		// Check if the parent directory exists, if not persist it first
		if dirPath != "/" && dirPath != "." && !p.PathExists(dirPath) {
			// Recursively persist the parent directory
			if err := p.PersistFile(dirPath, true); err != nil {
				return fmt.Errorf("failed to persist parent directory %s: %v", dirPath, err)
			}
		}

		// Remove existing entry for the path
		if err := tx.Where("path = ?", absPath).Delete(&FileSystem{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing data for path %s: %v", absPath, err)
		}

		// Insert or update the file or directory itself
		fs := FileSystem{
			Name:        filepath.Base(absPath),
			Path:        absPath,
			ParentID:    p.getParentID(dirPath),
			IsDirectory: isDir,
		}
		if err := tx.Save(&fs).Error; err != nil {
			return fmt.Errorf("failed to insert or update data for path %s: %v", absPath, err)
		}

		return nil
	})
}

func (p *GormPersistor) RemovePersistedFile(path string) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		// First, find the parent ID of the path to delete
		var parentID uint
		if err := tx.Model(&FileSystem{}).Select("parent_id").Where("path = ?", path).Scan(&parentID).Error; err != nil {
			return fmt.Errorf("failed to find parent ID: %v", err)
		}

		// Delete the file and its child entries
		if err := tx.Where("path = ? OR parent_id = ?", path, parentID).Delete(&FileSystem{}).Error; err != nil {
			return fmt.Errorf("failed to delete persisted data: %v", err)
		}

		return nil
	})
}

func (p *GormPersistor) LoadDirMap(path string) (dirMap map[string][]string, err error) {
	var fsRecords []FileSystem
	if err := p.db.Where("path LIKE ?", path+"%").Find(&fsRecords).Error; err != nil {
		return nil, err
	}

	dirMap = make(map[string][]string)
	for _, fs := range fsRecords {
		if fs.IsDirectory {
			dirMap[fs.Path] = []string{}
		}
		dirPath := filepath.Dir(fs.Path)
		if _, exists := dirMap[dirPath]; exists {
			dirMap[dirPath] = append(dirMap[dirPath], fs.Name)
		}
	}

	return dirMap, nil
}

func (p *GormPersistor) UpdatePaths(srcPath, dstPath string) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		// Update the paths for all records affected by the move operation
		if err := tx.Model(&FileSystem{}).Where("path LIKE ?", srcPath+"%").
			Update("path", gorm.Expr("REPLACE(path, ?, ?)", srcPath, dstPath)).Error; err != nil {
			return fmt.Errorf("failed to update paths: %v", err)
		}
		return nil
	})
}

// PathExists checks if a given path already exists in the database.
func (p *GormPersistor) PathExists(path string) bool {
	var count int64
	err := p.db.Model(&FileSystem{}).Where("path = ?", filepath.Clean(path)).Count(&count).Error
	return err == nil && count > 0
}

// Helper methods
func (p *GormPersistor) getParentID(path string) uint {
	parentPath := filepath.Dir(path)
	var fs FileSystem
	if err := p.db.Where("path = ?", parentPath).First(&fs).Error; err != nil {
		return 0 // No parent found, or error occurred
	}
	return fs.ID
}
