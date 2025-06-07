package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return NewConfigError("open source file", src, err)
	}
	defer srcFile.Close()
	
	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return NewConfigError("stat source file", src, err)
	}
	
	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return NewConfigError("create destination file", dst, err)
	}
	defer dstFile.Close()
	
	// Copy file contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return NewConfigError("copy file contents", src, err)
	}
	
	// Set permissions to match source
	if err := dstFile.Chmod(srcInfo.Mode()); err != nil {
		return NewConfigError("set file permissions", dst, err)
	}
	
	return nil
}

// copyDirectory recursively copies a directory from src to dst
func copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return NewConfigError("stat source directory", src, err)
	}
	
	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return NewConfigError("create destination directory", dst, err)
	}
	
	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return NewConfigError("read source directory", src, err)
	}
	
	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		
		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// moveFile moves a file from src to dst (rename with cross-filesystem support)
func moveFile(src, dst string) error {
	// Try simple rename first (works if on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	
	// If rename fails, copy and delete
	if err := copyFile(src, dst); err != nil {
		return err
	}
	
	// Remove source file
	if err := os.Remove(src); err != nil {
		return NewConfigError("remove source file", src, err)
	}
	
	return nil
}

// ensureDir creates directory if it doesn't exist
func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewConfigError("create directory", dir, err)
	}
	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isEmptyDir checks if directory is empty
func isEmptyDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, NewConfigError("read directory", dir, err)
	}
	return len(entries) == 0, nil
}

// removeEmptyDirs removes empty directories recursively up the path
func removeEmptyDirs(path string) error {
	for {
		// Don't remove if it's a root directory or important system directory
		if path == "/" || path == "." || path == ".." {
			break
		}
		
		isEmpty, err := isEmptyDir(path)
		if err != nil || !isEmpty {
			break
		}
		
		if err := os.Remove(path); err != nil {
			break
		}
		
		// Move up one directory level
		path = filepath.Dir(path)
	}
	
	return nil
}

// createTempFile creates a temporary file in the same directory as the target
func createTempFile(targetPath string) (*os.File, error) {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	
	tempFile, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return nil, NewConfigError("create temp file", targetPath, err)
	}
	
	return tempFile, nil
}

// atomicWrite writes data to a file atomically by writing to a temp file first
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	// Create temp file in same directory
	tempFile, err := createTempFile(path)
	if err != nil {
		return err
	}
	
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Clean up temp file if something goes wrong
	}()
	
	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return NewConfigError("write temp file", tempPath, err)
	}
	
	// Set permissions
	if err := tempFile.Chmod(perm); err != nil {
		return NewConfigError("set temp file permissions", tempPath, err)
	}
	
	// Close temp file
	if err := tempFile.Close(); err != nil {
		return NewConfigError("close temp file", tempPath, err)
	}
	
	// Atomically replace target file
	if err := os.Rename(tempPath, path); err != nil {
		return NewConfigError("replace target file", path, err)
	}
	
	return nil
}

// getSafeBackupPath generates a safe backup path that doesn't conflict
func getSafeBackupPath(originalPath string) string {
	base := originalPath + ".backup"
	counter := 1
	
	for fileExists(base) {
		base = originalPath + fmt.Sprintf(".backup.%d", counter)
		counter++
	}
	
	return base
}

// createBackupFile creates a backup of the given file
func createBackupFile(filePath string) (string, error) {
	if !fileExists(filePath) {
		return "", nil // No backup needed if file doesn't exist
	}
	
	backupPath := getSafeBackupPath(filePath)
	
	if err := copyFile(filePath, backupPath); err != nil {
		return "", NewConfigError("create backup", filePath, err)
	}
	
	return backupPath, nil
}
