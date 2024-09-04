package ufs

import (
	"fmt"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

// UserFileSystem represents an in-memory file system with a current working directory.
type UserFileSystem struct {
	fs      afero.Fs
	cwd     string // Current working directory
	fsMutex sync.RWMutex

	persistor Persistor
	dirMap    map[string][]string
}

// NewUserFileSystem creates a new UserFileSystem instance with an in-memory filesystem.
func NewUserFileSystem(db *gorm.DB) *UserFileSystem {
	fs := afero.NewMemMapFs()
	ufs := &UserFileSystem{
		fs:        fs,
		cwd:       "/",
		persistor: NewGormPersistor(db),
		dirMap:    make(map[string][]string),
	}

	// Restore the file system state from the database
	ufs.persistor.LoadDirMap("/")

	return ufs
}

// persistFile persists the metadata of a specific file or directory to disk using gob encoding.
func (ufs *UserFileSystem) persistFile(path string) error {
	isDir, err := ufs.IsDir(path)
	if err != nil {
		return err
	}
	return ufs.persistor.PersistFile(path, isDir)
}

// ReadFile reads the contents of a file.
func (ufs *UserFileSystem) ReadFile(name string) ([]byte, error) {
	absPath := ufs.resolvePath(name)
	return afero.ReadFile(ufs.fs, absPath)
}

// WriteFile writes data to a file.
func (ufs *UserFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	absPath := ufs.resolvePath(name)
	err := afero.WriteFile(ufs.fs, absPath, data, perm)
	if err != nil {
		return err
	}

	// Persist the changes for the parent directory
	if err := ufs.persistFile(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("failed to persist directory data: %v", err)
	}
	return nil
}

// Mv moves or renames a file or directory and updates the in-memory directory map.
func (ufs *UserFileSystem) Mv(src, dst string) error {
	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()
	srcPath := ufs.resolvePath(src)
	dstPath := ufs.resolvePath(dst)

	// Rename the file or directory on the filesystem
	err := ufs.fs.Rename(srcPath, dstPath)
	if err != nil {
		return err
	}

	// Update the in-memory directory map
	srcDir := filepath.Dir(srcPath)
	dstDir := filepath.Dir(dstPath)
	srcBase := filepath.Base(srcPath)
	dstBase := filepath.Base(dstPath)

	// Remove the entry from the source directory
	entries := ufs.dirMap[srcDir]
	var newEntries []string
	for _, entry := range entries {
		if entry != srcBase {
			newEntries = append(newEntries, entry)
		}
	}
	if len(newEntries) > 0 {
		ufs.dirMap[srcDir] = newEntries
	} else {
		delete(ufs.dirMap, srcDir)
	}

	// Add the entry to the destination directory
	ufs.dirMap[dstDir] = append(ufs.dirMap[dstDir], dstBase)

	// Persist changes to the source and destination directories
	if err := ufs.persistFile(srcDir); err != nil {
		return fmt.Errorf("failed to persist source directory data: %v", err)
	}
	if err := ufs.persistFile(dstDir); err != nil {
		return fmt.Errorf("failed to persist destination directory data: %v", err)
	}

	// Remove any previous entry in the destination path, as it should not be listed twice
	delete(ufs.dirMap, dstPath)
	return nil
}

// Ls lists the contents of the specified directory or the current working directory if path is empty.
func (ufs *UserFileSystem) Ls(path string) ([]string, error) {
	dirPath := ufs.resolvePath(path)

	ufs.fsMutex.RLock()
	defer ufs.fsMutex.RUnlock()

	entries, ok := ufs.dirMap[dirPath]
	if !ok {
		return nil, fmt.Errorf("directory not found: %s", dirPath)
	}

	return entries, nil
}

// Mkdir creates a new directory and all necessary parent directories using afero's MkdirAll.
func (ufs *UserFileSystem) Mkdir(path string, perm os.FileMode) error {
	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	absPath := ufs.resolvePath(path)

	// Use afero's MkdirAll to create the directory and its parents
	if err := ufs.fs.MkdirAll(absPath, perm); err != nil {
		return err
	}

	// Update the in-memory directory map
	dirPath := filepath.Dir(absPath)
	ufs.dirMap[dirPath] = append(ufs.dirMap[dirPath], filepath.Base(absPath))
	ufs.dirMap[absPath] = []string{} // Initialize the new directory

	// Persist the parent directory first
	if err := ufs.persistFile(dirPath); err != nil {
		return fmt.Errorf("failed to persist parent directory %s: %v", dirPath, err)
	}

	// Persist the new directory
	if err := ufs.persistFile(absPath); err != nil {
		return fmt.Errorf("failed to persist directory %s: %v", absPath, err)
	}

	return nil
}

// Create creates a new file and updates the in-memory directory map.
func (ufs *UserFileSystem) Create(name string) (afero.File, error) {
	absPath := ufs.resolvePath(name)
	file, err := ufs.fs.Create(absPath)
	if err != nil {
		return nil, err
	}

	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	// Update the in-memory directory map
	dirPath := filepath.Dir(absPath)
	ufs.dirMap[dirPath] = append(ufs.dirMap[dirPath], filepath.Base(absPath))

	// Persist the changes for the parent directory
	return file, ufs.persistFile(dirPath)
}

// Remove removes a file or directory.
func (ufs *UserFileSystem) Remove(name string) error {
	absPath := ufs.resolvePath(name)

	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	// Check if the path exists
	info, err := ufs.fs.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// Use a stack-based approach to delete directories iteratively
		stack := []string{absPath}
		for len(stack) > 0 {
			dir := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			entries, err := afero.ReadDir(ufs.fs, dir)
			if err != nil {
				return err
			}

			// Collect all entries to remove
			for _, entry := range entries {
				entryPath := filepath.Join(dir, entry.Name())
				if entry.IsDir() {
					stack = append(stack, entryPath)
				}
				if err := ufs.fs.Remove(entryPath); err != nil {
					return err
				}

				// Remove from in-memory directory map
				dirPath := filepath.Dir(entryPath)
				if entries := ufs.dirMap[dirPath]; entries != nil {
					newEntries := []string{}
					for _, e := range entries {
						if e != filepath.Base(entryPath) {
							newEntries = append(newEntries, e)
						}
					}
					if len(newEntries) > 0 {
						ufs.dirMap[dirPath] = newEntries
					} else {
						delete(ufs.dirMap, dirPath)
					}
				}

				// Remove the persisted data
				if err := ufs.persistor.RemovePersistedFile(entryPath); err != nil {
					return err
				}
			}

			// Remove the now-empty directory itself
			if err := ufs.fs.Remove(dir); err != nil {
				return err
			}

			// Remove the directory from the in-memory map
			dirPath := filepath.Dir(dir)
			if entries := ufs.dirMap[dirPath]; entries != nil {
				newEntries := []string{}
				for _, e := range entries {
					if e != filepath.Base(dir) {
						newEntries = append(newEntries, e)
					}
				}
				if len(newEntries) > 0 {
					ufs.dirMap[dirPath] = newEntries
				} else {
					delete(ufs.dirMap, dirPath)
				}
			}
		}
	} else {
		// Remove a file
		if err := ufs.fs.Remove(absPath); err != nil {
			return err
		}

		// Update the in-memory directory map
		dirPath := filepath.Dir(absPath)
		entries := ufs.dirMap[dirPath]
		newEntries := []string{}
		for _, entry := range entries {
			if entry != filepath.Base(absPath) {
				newEntries = append(newEntries, entry)
			}
		}

		ufs.dirMap[dirPath] = newEntries

		// Remove the file from the in-memory map
		delete(ufs.dirMap, absPath)

		// Remove the persisted data
		if err := ufs.persistor.RemovePersistedFile(name); err != nil {
			return err
		}
	}

	return nil
}

// resolvePath resolves a relative path to an absolute path based on the current working directory.
func (ufs *UserFileSystem) resolvePath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(ufs.cwd, path)
	}
	return filepath.Clean(path)
}

// IsDir checks if the given path is a directory.
func (ufs *UserFileSystem) IsDir(path string) (bool, error) {
	absPath := ufs.resolvePath(path)
	info, err := ufs.fs.Stat(absPath)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
