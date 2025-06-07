package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Operation represents a single atomic operation that can be rolled back
type Operation interface {
	Execute() error
	Rollback() error
	Description() string
	GetFile() string
}

// Transaction manages a group of operations that should be executed atomically
type Transaction struct {
	operations []Operation
	executed   []Operation // Successfully executed operations (for rollback)
	id         string
}

// NewTransaction creates a new transaction
func NewTransaction() *Transaction {
	return &Transaction{
		operations: make([]Operation, 0),
		executed:   make([]Operation, 0),
		id:         fmt.Sprintf("tx_%d", time.Now().Unix()),
	}
}

// AddOperation adds an operation to the transaction
func (t *Transaction) AddOperation(op Operation) {
	t.operations = append(t.operations, op)
}

// Execute runs all operations in the transaction
// If any operation fails, all successfully executed operations are rolled back
func (t *Transaction) Execute() error {
	var multiErr MultiError
	multiErr.Op = fmt.Sprintf("transaction %s", t.id)
	
	for i, op := range t.operations {
		if err := op.Execute(); err != nil {
			// Operation failed, rollback all previous operations
			rollbackErr := t.rollback()
			if rollbackErr != nil {
				multiErr.Add(fmt.Errorf("operation %d failed: %v; rollback also failed: %v", i, err, rollbackErr))
			} else {
				multiErr.Add(fmt.Errorf("operation %d failed: %v (rolled back successfully)", i, err))
			}
			
			if multiErr.HasErrors() {
				return &multiErr
			}
			return NewConfigError("transaction execute", fmt.Sprintf("operation %d", i), err)
		}
		
		// Track successfully executed operations for potential rollback
		t.executed = append(t.executed, op)
	}
	
	return nil
}

// rollback undoes all successfully executed operations in reverse order
func (t *Transaction) rollback() error {
	var multiErr MultiError
	multiErr.Op = fmt.Sprintf("rollback transaction %s", t.id)
	
	// Rollback in reverse order
	for i := len(t.executed) - 1; i >= 0; i-- {
		if err := t.executed[i].Rollback(); err != nil {
			multiErr.Add(fmt.Errorf("failed to rollback operation %d (%s): %v", 
				i, t.executed[i].Description(), err))
		}
	}
	
	// Clear executed operations
	t.executed = t.executed[:0]
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}

// Rollback manually rolls back the transaction (useful for testing or explicit rollback)
func (t *Transaction) Rollback() error {
	return t.rollback()
}

// GetOperations returns a copy of the operations list
func (t *Transaction) GetOperations() []Operation {
	ops := make([]Operation, len(t.operations))
	copy(ops, t.operations)
	return ops
}

// LinkOperation handles creating a symlink with backup
type LinkOperation struct {
	sourcePath string
	targetPath string
	backupPath string
	created    bool
	backed     bool
	file       *ConfigFile
}

// NewLinkOperation creates a new link operation
func NewLinkOperation(sourcePath, targetPath string, file *ConfigFile) *LinkOperation {
	return &LinkOperation{
		sourcePath: sourcePath,
		targetPath: targetPath,
		file:       file,
	}
}

func (op *LinkOperation) Execute() error {
	// Check if target already exists
	if _, err := os.Lstat(op.targetPath); err == nil {
		// Target exists, create backup
		op.backupPath = op.targetPath + ".backup." + time.Now().Format("20060102-150405")
		if err := os.Rename(op.targetPath, op.backupPath); err != nil {
			return NewConfigError("backup existing file", op.targetPath, err)
		}
		op.backed = true
	}
	
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(op.targetPath), 0755); err != nil {
		return NewConfigError("create target directory", filepath.Dir(op.targetPath), err)
	}
	
	// Create symlink
	if err := os.Symlink(op.sourcePath, op.targetPath); err != nil {
		return NewConfigError("create symlink", op.targetPath, err)
	}
	
	op.created = true
	return nil
}

func (op *LinkOperation) Rollback() error {
	var multiErr MultiError
	multiErr.Op = "rollback link operation"
	
	// Remove symlink if we created it
	if op.created {
		if err := os.Remove(op.targetPath); err != nil && !os.IsNotExist(err) {
			multiErr.Add(NewConfigError("remove symlink", op.targetPath, err))
		}
	}
	
	// Restore backup if we created one
	if op.backed && op.backupPath != "" {
		if err := os.Rename(op.backupPath, op.targetPath); err != nil {
			multiErr.Add(NewConfigError("restore backup", op.backupPath, err))
		}
	}
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}

func (op *LinkOperation) Description() string {
	return fmt.Sprintf("link %s -> %s", op.targetPath, op.sourcePath)
}

func (op *LinkOperation) GetFile() string {
	if op.file != nil {
		return op.file.Name
	}
	return filepath.Base(op.targetPath)
}

// CopyOperation handles copying files/directories with backup
type CopyOperation struct {
	sourcePath string
	targetPath string
	backupPath string
	copied     bool
	backed     bool
	isDir      bool
	file       *ConfigFile
}

// NewCopyOperation creates a new copy operation
func NewCopyOperation(sourcePath, targetPath string, file *ConfigFile) *CopyOperation {
	isDir := false
	if info, err := os.Stat(sourcePath); err == nil {
		isDir = info.IsDir()
	}
	
	return &CopyOperation{
		sourcePath: sourcePath,
		targetPath: targetPath,
		isDir:      isDir,
		file:       file,
	}
}

func (op *CopyOperation) Execute() error {
	// Check if target already exists
	if _, err := os.Lstat(op.targetPath); err == nil {
		// Target exists, create backup
		op.backupPath = op.targetPath + ".backup." + time.Now().Format("20060102-150405")
		if err := os.Rename(op.targetPath, op.backupPath); err != nil {
			return NewConfigError("backup existing file", op.targetPath, err)
		}
		op.backed = true
	}
	
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(op.targetPath), 0755); err != nil {
		return NewConfigError("create target directory", filepath.Dir(op.targetPath), err)
	}
	
	// Handle case where source is empty (create basic file)
	if op.sourcePath == "" {
		// Create a basic config file
		basicContent := fmt.Sprintf("# %s configuration\n# Generated by config-manager\n# Please customize as needed\n", 
			filepath.Base(op.targetPath))
		if err := os.WriteFile(op.targetPath, []byte(basicContent), 0644); err != nil {
			return NewConfigError("create basic file", op.targetPath, err)
		}
		op.copied = true
		return nil
	}
	
	// Copy file or directory
	var err error
	if op.isDir {
		err = copyDirectory(op.sourcePath, op.targetPath)
	} else {
		err = copyFile(op.sourcePath, op.targetPath)
	}
	
	if err != nil {
		return NewConfigError("copy file", op.sourcePath, err)
	}
	
	op.copied = true
	return nil
}

func (op *CopyOperation) Rollback() error {
	var multiErr MultiError
	multiErr.Op = "rollback copy operation"
	
	// Remove copied file/directory if we created it
	if op.copied {
		if err := os.RemoveAll(op.targetPath); err != nil && !os.IsNotExist(err) {
			multiErr.Add(NewConfigError("remove copied file", op.targetPath, err))
		}
	}
	
	// Restore backup if we created one
	if op.backed && op.backupPath != "" {
		if err := os.Rename(op.backupPath, op.targetPath); err != nil {
			multiErr.Add(NewConfigError("restore backup", op.backupPath, err))
		}
	}
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}

func (op *CopyOperation) Description() string {
	if op.isDir {
		return fmt.Sprintf("copy directory %s -> %s", op.sourcePath, op.targetPath)
	}
	return fmt.Sprintf("copy file %s -> %s", op.sourcePath, op.targetPath)
}

func (op *CopyOperation) GetFile() string {
	if op.file != nil {
		return op.file.Name
	}
	return filepath.Base(op.targetPath)
}

// TemplateOperation handles template processing
type TemplateOperation struct {
	config       *Config
	file         *ConfigFile
	templatePath string
	outputPath   string
	created      bool
	backupPath   string
	backed       bool
}

// NewTemplateOperation creates a new template operation
func NewTemplateOperation(config *Config, file *ConfigFile, templatePath, outputPath string) *TemplateOperation {
	return &TemplateOperation{
		config:       config,
		file:         file,
		templatePath: templatePath,
		outputPath:   outputPath,
	}
}

func (op *TemplateOperation) Execute() error {
	// Check if output already exists
	if _, err := os.Lstat(op.outputPath); err == nil {
		// Output exists, create backup
		op.backupPath = op.outputPath + ".backup." + time.Now().Format("20060102-150405")
		if err := os.Rename(op.outputPath, op.backupPath); err != nil {
			return NewConfigError("backup existing template output", op.outputPath, err)
		}
		op.backed = true
	}
	
	// Process template
	if err := createFromTemplate(op.config, op.file, op.outputPath); err != nil {
		return err
	}
	
	op.created = true
	return nil
}

func (op *TemplateOperation) Rollback() error {
	var multiErr MultiError
	multiErr.Op = "rollback template operation"
	
	// Remove created file if we created it
	if op.created {
		if err := os.Remove(op.outputPath); err != nil && !os.IsNotExist(err) {
			multiErr.Add(NewConfigError("remove template output", op.outputPath, err))
		}
	}
	
	// Restore backup if we created one
	if op.backed && op.backupPath != "" {
		if err := os.Rename(op.backupPath, op.outputPath); err != nil {
			multiErr.Add(NewConfigError("restore backup", op.backupPath, err))
		}
	}
	
	if multiErr.HasErrors() {
		return &multiErr
	}
	
	return nil
}

func (op *TemplateOperation) Description() string {
	return fmt.Sprintf("process template %s -> %s", op.templatePath, op.outputPath)
}

func (op *TemplateOperation) GetFile() string {
	return op.file.Name
}

// Helper function to create atomic link operation for a config file
func createAtomicLinkOperation(config *Config, file *ConfigFile) (*Transaction, error) {
	tx := NewTransaction()
	
	sourceDir := filepath.Dir(filepath.Join(config.DotfilesDir, file.Source))
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return nil, NewConfigError("create source directory", sourceDir, err)
	}
	
	sourcePath := filepath.Join(config.DotfilesDir, file.Source)
	
	// If source doesn't exist and it's a template, create from template first
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		if file.Template {
			templatePath := findTemplateFile(config, file.Name, file.Source, file.Category)
			if templatePath != "" {
				// Add template operation
				templateOp := NewTemplateOperation(config, file, templatePath, sourcePath)
				tx.AddOperation(templateOp)
			} else {
				// Create basic file operation
				copyOp := NewCopyOperation("", sourcePath, file) // Empty source means create basic file
				tx.AddOperation(copyOp)
			}
		} else {
			// For non-templates, we might want to copy existing file if it exists
			if _, err := os.Stat(file.Target); err == nil {
				// Target exists, copy it to source first
				copyOp := NewCopyOperation(file.Target, sourcePath, file)
				tx.AddOperation(copyOp)
			}
		}
	}
	
	// Add link operation
	linkOp := NewLinkOperation(sourcePath, file.Target, file)
	tx.AddOperation(linkOp)
	
	return tx, nil
}

// atomicLinkAllConfigs creates atomic transactions for linking all configs
func atomicLinkAllConfigs(config *Config) error {
	var allResults []OperationResult
	var failedFiles []string
	
	for _, file := range config.Files {
		tx, err := createAtomicLinkOperation(config, &file)
		if err != nil {
			result := OperationResult{
				File:    file.Name,
				Success: false,
				Message: "Failed to create transaction",
				Error:   err,
			}
			allResults = append(allResults, result)
			failedFiles = append(failedFiles, file.Name)
			continue
		}
		
		if err := tx.Execute(); err != nil {
			result := OperationResult{
				File:    file.Name,
				Success: false,
				Message: "Transaction failed",
				Error:   err,
			}
			allResults = append(allResults, result)
			failedFiles = append(failedFiles, file.Name)
		} else {
			result := OperationResult{
				File:    file.Name,
				Success: true,
				Message: "Successfully linked",
			}
			allResults = append(allResults, result)
		}
	}
	
	// If any files failed, return error with details
	if len(failedFiles) > 0 {
		var multiErr MultiError
		multiErr.Op = "atomic link all configs"
		for _, result := range allResults {
			if !result.Success {
				multiErr.Add(fmt.Errorf("%s: %v", result.File, result.Error))
			}
		}
		return &multiErr
	}
	
	return nil
}

// atomicLinkSingleConfig creates and executes atomic transaction for a single config
func atomicLinkSingleConfig(config *Config, file *ConfigFile) error {
	tx, err := createAtomicLinkOperation(config, file)
	if err != nil {
		return NewConfigError("create transaction", file.Name, err)
	}
	
	return tx.Execute()
}
