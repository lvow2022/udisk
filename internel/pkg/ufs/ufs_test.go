package ufs

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestUfs(t *testing.T) {
	// Open an SQLite database in memory for testing purposes
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{}) // Use in-memory database for faster tests
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	err = db.AutoMigrate(&FileSystem{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	fs := NewUserFileSystem(db)

	// Test MkdirAll
	if err := fs.Mkdir("/testdir/parent/subdir", 0755); err != nil {
		t.Fatalf("Error creating directory: %v", err)
	}

	// Test Create
	file, err := fs.Create("/testdir/parent/subdir/file1.txt")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}
	file.Close() // Ensure file is closed after creation

	// Test WriteFile
	if err := fs.WriteFile("/testdir/parent/subdir/file1.txt", []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Error writing file: %v", err)
	}

	// Test ReadFile
	data, err := fs.ReadFile("/testdir/parent/subdir/file1.txt")
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	if string(data) != "Hello, World!" {
		t.Fatalf("File contents mismatch: expected 'Hello, World!', got '%s'", string(data))
	}

	// Test Ls
	files, err := fs.Ls("/testdir/parent/subdir")
	if err != nil {
		t.Fatalf("Error listing directory: %v", err)
	}
	if len(files) != 1 || files[0] != "file1.txt" {
		t.Fatalf("Directory contents mismatch: expected ['file1.txt'], got %v", files)
	}

	// Test Mv (Move/Rename)
	if err := fs.Mv("/testdir/parent/subdir/file1.txt", "/testdir/parent/subdir/file2.txt"); err != nil {
		t.Fatalf("Error moving file: %v", err)
	}

	// Verify file move
	files, err = fs.Ls("/testdir/parent/subdir")
	if err != nil {
		t.Fatalf("Error listing directory after move: %v", err)
	}
	if len(files) != 1 || files[0] != "file2.txt" {
		t.Fatalf("Directory contents mismatch after move: expected ['file2.txt'], got %v", files)
	}

	// Test Remove (file)
	if err := fs.Remove("/testdir/parent/subdir/file2.txt"); err != nil {
		t.Fatalf("Error removing file: %v", err)
	}

	// Verify file removal
	files, err = fs.Ls("/testdir/parent/subdir")
	if err != nil {
		t.Fatalf("Error listing directory after file removal: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("Directory contents mismatch after file removal: expected [], got %v", files)
	}

	// Test Remove (directory)
	if err := fs.Remove("/testdir/parent/subdir"); err != nil {
		t.Fatalf("Error removing directory: %v", err)
	}

	// Verify directory removal
	files, err = fs.Ls("/testdir/parent")
	//if err != nil {
	//	t.Fatalf("Error listing directory after directory removal: %v", err)
	//}
	if len(files) != 0 {
		t.Fatalf("Directory contents mismatch after directory removal: expected [], got %v", files)
	}

	// Test edge case: removing a non-existent file
	err = fs.Remove("/nonexistent/file.txt")
	if err == nil {
		t.Fatalf("Expected error when removing a non-existent file, got nil")
	}

	// Test edge case: listing a non-existent directory
	_, err = fs.Ls("/nonexistent/dir")
	if err == nil {
		t.Fatalf("Expected error when listing a non-existent directory, got nil")
	}

	fmt.Println("All tests passed successfully!")
}
