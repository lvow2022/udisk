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
}

type FileSystem struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Path        string `gorm:"uniqueIndex"`
	ParentID    uint
	IsDirectory bool
	Content     []byte      `gorm:"type:blob"`
	Parent      *FileSystem `gorm:"foreignKey:ParentID"`
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
		if err := tx.Where("path = ?", path).Or("parent_id = (SELECT id FROM file_system WHERE path = ?)", path).Delete(&FileSystem{}).Error; err != nil {
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

// Helper methods
func (p *GormPersistor) getParentID(path string) uint {
	parentPath := filepath.Dir(path)
	var fs FileSystem
	if err := p.db.Where("path = ?", parentPath).First(&fs).Error; err != nil {
		return 0 // No parent found, or error occurred
	}
	return fs.ID
}
