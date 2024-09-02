package ufs

import (
	"fmt"
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"sync"
)

// uFileSystem represents an in-memory file system with a current working directory.
type uFileSystem struct {
	fs      afero.Fs
	cwd     string // Current working directory
	fsMutex sync.RWMutex
	// In-memory directory structure: path -> list of entries (files and directories)
	dirMap map[string][]string
}

// NewUFileSystem creates a new uFileSystem instance with an in-memory filesystem.
func NewUFileSystem() *uFileSystem {
	fs := afero.NewMemMapFs()
	return &uFileSystem{
		fs:     fs,
		cwd:    "/",
		dirMap: make(map[string][]string),
	}
}

// Pwd returns the current working directory.
func (ufs *uFileSystem) Pwd() string {
	return ufs.cwd
}

// Cd changes the current working directory.
func (ufs *uFileSystem) Cd(path string) error {
	newPath := ufs.resolvePath(path)

	// Check if the directory exists and is a directory
	info, err := ufs.fs.Stat(newPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", newPath)
	}

	ufs.cwd = newPath
	return nil
}

// Mv moves or renames a file or directory and updates the in-memory directory map.
func (ufs *uFileSystem) Mv(src, dst string) error {
	srcPath := ufs.resolvePath(src)
	dstPath := ufs.resolvePath(dst)

	// Rename the file or directory on the filesystem
	err := ufs.fs.Rename(srcPath, dstPath)
	if err != nil {
		return err
	}

	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	// Update the in-memory directory map
	srcDir := filepath.Dir(srcPath)
	dstDir := filepath.Dir(dstPath)
	srcBase := filepath.Base(srcPath)
	dstBase := filepath.Base(dstPath)

	// Remove the entry from the source directory
	if entries, ok := ufs.dirMap[srcDir]; ok {
		newEntries := []string{}
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
	}

	// Add the entry to the destination directory
	if entries, ok := ufs.dirMap[dstDir]; ok {
		ufs.dirMap[dstDir] = append(entries, dstBase)
	} else {
		ufs.dirMap[dstDir] = []string{dstBase}
	}

	// Remove any previous entry in the destination path, as it should not be listed twice
	delete(ufs.dirMap, dstPath)

	return nil
}

// Ls lists the contents of the specified directory or the current working directory if path is empty.
func (ufs *uFileSystem) Ls(path string) ([]string, error) {
	if path == "" {
		path = "."
	}
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
func (ufs *uFileSystem) Mkdir(path string, perm os.FileMode) error {
	absPath := ufs.resolvePath(path)

	// Use afero's MkdirAll to create the directory and its parents
	if err := ufs.fs.MkdirAll(absPath, perm); err != nil {
		return err
	}

	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	// Update the in-memory directory map
	dirPath := filepath.Dir(absPath)
	if dirPath == "." {
		dirPath = ufs.cwd
	}

	entries, exists := ufs.dirMap[dirPath]
	if !exists {
		entries = []string{}
	}
	ufs.dirMap[dirPath] = append(entries, filepath.Base(absPath))
	ufs.dirMap[absPath] = []string{} // Initialize the new directory
	return nil
}

// Create creates a new file and updates the in-memory directory map.
func (ufs *uFileSystem) Create(name string) (afero.File, error) {
	absPath := ufs.resolvePath(name)
	file, err := ufs.fs.Create(absPath)
	if err != nil {
		return nil, err
	}

	ufs.fsMutex.Lock()
	defer ufs.fsMutex.Unlock()

	// Update the in-memory directory map
	dirPath := filepath.Dir(absPath)
	if dirPath == "." {
		dirPath = ufs.cwd
	}

	entries, exists := ufs.dirMap[dirPath]
	if !exists {
		entries = []string{}
	}
	ufs.dirMap[dirPath] = append(entries, filepath.Base(absPath))
	return file, nil
}

// Remove removes a file or directory.
func (ufs *uFileSystem) Remove(name string) error {
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
				err = ufs.fs.Remove(entryPath)
				if err != nil {
					return err
				}
			}

			// Remove the now-empty directory itself
			err = ufs.fs.Remove(dir)
			if err != nil {
				return err
			}
		}
	} else {
		// Remove a file
		err = ufs.fs.Remove(absPath)
		if err != nil {
			return err
		}
	}

	// Update the in-memory directory map
	dirPath := filepath.Dir(absPath)
	if entries, ok := ufs.dirMap[dirPath]; ok {
		newEntries := []string{}
		for _, entry := range entries {
			if entry != filepath.Base(absPath) {
				newEntries = append(newEntries, entry)
			}
		}
		if len(newEntries) > 0 {
			ufs.dirMap[dirPath] = newEntries
		} else {
			delete(ufs.dirMap, dirPath)
		}
	}
	delete(ufs.dirMap, absPath)
	return nil
}

// ReadFile reads the contents of a file.
func (ufs *uFileSystem) ReadFile(name string) ([]byte, error) {
	absPath := ufs.resolvePath(name)
	return afero.ReadFile(ufs.fs, absPath)
}

// WriteFile writes data to a file.
func (ufs *uFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	absPath := ufs.resolvePath(name)
	return afero.WriteFile(ufs.fs, absPath, data, perm)
}

// resolvePath resolves a given path relative to the current working directory.
func (ufs *uFileSystem) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(ufs.cwd, path))
}

func testUfs() {
	fs := NewUFileSystem()

	// Test MkdirAll
	err := fs.Mkdir("/testdir/parent/subdir", 0755)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	// Test Create
	_, err = fs.Create("/testdir/parent/subdir/file1.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	// Test WriteFile
	err = fs.WriteFile("/testdir/parent/subdir/file1.txt", []byte("Hello, World!"), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	// Test ReadFile
	data, err := fs.ReadFile("/testdir/parent/subdir/file1.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Println("File contents:", string(data))

	// Test Ls
	files, err := fs.Ls("/testdir/parent/subdir")
	if err != nil {
		fmt.Println("Error listing directory:", err)
		return
	}
	fmt.Println("Contents of /testdir/parent/subdir:", files)

	// Test Remove
	err = fs.Remove("/testdir/parent/subdir/file1.txt")
	if err != nil {
		fmt.Println("Error removing file:", err)
		return
	}

	// Verify file removal
	files, err = fs.Ls("/testdir/parent/subdir")
	if err != nil {
		fmt.Println("Error listing directory:", err)
		return
	}
	fmt.Println("Contents of /testdir/parent/subdir after removal:", files)

	// Test Remove directory
	err = fs.Remove("/testdir/parent/subdir")
	if err != nil {
		fmt.Println("Error removing directory:", err)
		return
	}

	// Verify directory removal
	files, err = fs.Ls("/testdir/parent")
	if err != nil {
		fmt.Println("Error listing directory:", err)
		return
	}
	fmt.Println("Contents of /testdir/parent after removing subdir:", files)
}
