package pipeline

import (
	"fmt"
	"hermes-decompiler/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
)

const backupDirName = "node_modules_pristine"
const packageLockBackup = "package-lock.json.pristine"

// CreateBackup creates a pristine backup of node_modules and package-lock.json
func CreateBackup(workingDir string) error {
	backupPath := filepath.Join(workingDir, backupDirName)
	if _, err := os.Stat(backupPath); err == nil {
		utils.D.Println("Backup already exists, skipping creation.")
		return nil
	}

	utils.L.Printf("Creating pristine backup of node_modules in %s\n", workingDir)

	// Copy node_modules
	srcPath := filepath.Join(workingDir, "node_modules")
	cmd := exec.Command("cp", "-R", srcPath, backupPath)
	cmd.Dir = workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to backup node_modules: %w", err)
	}

	// Copy package-lock.json
	srcLockPath := filepath.Join(workingDir, "package-lock.json")
	destLockPath := filepath.Join(workingDir, packageLockBackup)
	cmd = exec.Command("cp", srcLockPath, destLockPath)
	cmd.Dir = workingDir
	if err := cmd.Run(); err != nil {
		// Cleanup partial backup if lock copy fails
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to backup package-lock.json: %w", err)
	}

	utils.S.Println("Pristine backup created successfully.")
	return nil
}

// RestoreFromBackup restores the pristine environment from the backup
func RestoreFromBackup(workingDir string) error {
	utils.L.Printf("Restoring pristine environment in %s\n", workingDir)
	backupPath := filepath.Join(workingDir, backupDirName)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory '%s' not found in '%s'", backupDirName, workingDir)
	}

	// Remove current node_modules
	currentModulesPath := filepath.Join(workingDir, "node_modules")
	if err := os.RemoveAll(currentModulesPath); err != nil {
		return fmt.Errorf("failed to remove existing node_modules: %w", err)
	}

	// Restore node_modules from backup
	cmd := exec.Command("cp", "-R", backupPath, currentModulesPath)
	cmd.Dir = workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore node_modules: %w", err)
	}

	// Restore package-lock.json
	srcLockPath := filepath.Join(workingDir, packageLockBackup)
	destLockPath := filepath.Join(workingDir, "package-lock.json")
	cmd = exec.Command("cp", srcLockPath, destLockPath)
	cmd.Dir = workingDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore package-lock.json: %w", err)
	}

	utils.S.Println("Pristine environment restored successfully.")
	return nil
}

// TODO: get all hermes compiler paths
func GetAllHermesCompilerPaths() []string {
	return []string{}
}

// TODO: get all hermes compiler versions
func GetAllHermesCompilerVersions() []string {
	return []string{}
}

// TODO: get all react native versions
func GetAllReactNativeVersions() []string {
	return []string{}
}
