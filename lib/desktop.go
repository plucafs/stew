package stew

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/marwanhawari/stew/constants"
	"golang.org/x/term"
)

// DesktopSupported reports whether the current environment supports creating .desktop files
func DesktopSupported() bool {
	return runtime.GOOS == "linux" && term.IsTerminal(int(os.Stdin.Fd()))
}

// GetDesktopApplicationsDir returns the XDG applications directory for the current user
func GetDesktopApplicationsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	xdgDataHomePath := os.Getenv("XDG_DATA_HOME")
	if xdgDataHomePath == "" {
		return filepath.Join(homeDir, ".local", "share", "applications"), nil
	}

	return filepath.Join(xdgDataHomePath, "applications"), nil
}

// DesktopFileName returns the stew-prefixed .desktop file name for an installed binary
func DesktopFileName(binaryName string) string {
	return "stew-" + binaryName + ".desktop"
}

func sanitizeDesktopValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func quoteExecPath(path string) string {
	escapedPath := strings.ReplaceAll(path, `\`, `\\`)
	escapedPath = strings.ReplaceAll(escapedPath, `"`, `\"`)
	return `"` + escapedPath + `"`
}

func buildDesktopEntry(binaryName, binaryPath, comment string) string {
	lines := []string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=" + sanitizeDesktopValue(binaryName),
	}
	comment = sanitizeDesktopValue(comment)
	if comment != "" {
		lines = append(lines, "Comment="+comment)
	}
	lines = append(lines,
		"Exec="+quoteExecPath(binaryPath),
		"Icon=application-x-designer",
		"Terminal=false",
		"Categories=Utility;",
	)
	return strings.Join(lines, "\n") + "\n"
}

func updateNameAndExec(desktopEntry, newBinaryName, newBinaryPath string) string {
	lines := strings.Split(desktopEntry, "\n")
	inDesktopEntryGroup := true
	for index, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			inDesktopEntryGroup = trimmedLine == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntryGroup {
			continue
		}
		if strings.HasPrefix(line, "Name=") {
			lines[index] = "Name=" + sanitizeDesktopValue(newBinaryName)
		} else if strings.HasPrefix(line, "Exec=") {
			lines[index] = "Exec=" + quoteExecPath(newBinaryPath)
		}
	}
	return strings.Join(lines, "\n")
}

func updateDesktopDatabase(applicationsDir string) {
	updateDesktopDatabasePath, err := exec.LookPath("update-desktop-database")
	if err != nil {
		return
	}
	cmd := exec.Command(updateDesktopDatabasePath, applicationsDir)
	_ = cmd.Run()
}

// CreateDesktopFile writes a new .desktop file and will never overwrite an existing one
func CreateDesktopFile(desktopFilePath, binaryName, binaryPath, comment string) error {
	if err := os.MkdirAll(filepath.Dir(desktopFilePath), 0755); err != nil {
		return err
	}

	desktopEntry := buildDesktopEntry(binaryName, binaryPath, comment)
	desktopFile, err := os.OpenFile(desktopFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer desktopFile.Close()

	_, err = desktopFile.WriteString(desktopEntry)
	return err
}

// SetupDesktopFile prompts you to create a .desktop file after a successful install.
// An existing .desktop file is never overwritten, and exiting the prompt counts as a decline
// so that the already-completed install is never left without a lockfile entry.
// It returns the name of the created .desktop file (empty if none was created) and
// whether an existing .desktop file was found (in which case the lockfile is left unchanged).
func SetupDesktopFile(binaryName, binaryPath string, getComment func() string) (string, bool) {
	if !DesktopSupported() {
		return "", false
	}

	applicationsDir, err := GetDesktopApplicationsDir()
	if err != nil {
		fmt.Printf("%v Could not resolve the desktop applications directory: %v\n", constants.YellowColor("WARNING:"), err)
		return "", false
	}

	desktopFilePath := filepath.Join(applicationsDir, DesktopFileName(binaryName))
	desktopFileExists, err := PathExists(desktopFilePath)
	if err != nil {
		fmt.Printf("%v Could not check for an existing desktop file: %v\n", constants.YellowColor("WARNING:"), err)
		return "", false
	}
	if desktopFileExists {
		fmt.Printf("ℹ️ Desktop file already exists, leaving it untouched: %v\n", constants.GreenColor(desktopFilePath))
		return "", true
	}

	userCreatingDesktopFile, err := WarningPromptConfirm(fmt.Sprintf("Create a .desktop file for %v?", constants.YellowColor(binaryName)))
	if err != nil || !userCreatingDesktopFile {
		return "", false
	}

	comment := ""
	if getComment != nil {
		comment = getComment()
	}

	if err := CreateDesktopFile(desktopFilePath, binaryName, binaryPath, comment); err != nil {
		if desktopFileAlreadyExists, _ := PathExists(desktopFilePath); desktopFileAlreadyExists {
			fmt.Printf("ℹ️ Desktop file already exists, leaving it untouched: %v\n", constants.GreenColor(desktopFilePath))
			return "", true
		}
		fmt.Printf("%v Could not create the desktop file %v: %v\n", constants.YellowColor("WARNING:"), constants.YellowColor(desktopFilePath), err)
		return "", false
	}
	fmt.Printf("📄 Created %v\n", constants.GreenColor(desktopFilePath))
	updateDesktopDatabase(applicationsDir)

	return DesktopFileName(binaryName), false
}

// RenameDesktopFile surgically updates the Name and Exec values of a tracked .desktop file
// when the binary is renamed. Any other content, like a hand-edited Icon value, is preserved.
// It will never overwrite an existing .desktop file.
// It returns the .desktop file name to record in the lockfile, or an empty string if the
// tracked .desktop file is gone and the lockfile field should be cleared.
func RenameDesktopFile(oldDesktopFileName, newBinaryName, newBinaryPath string) (string, error) {
	if oldDesktopFileName == "" {
		return "", nil
	}

	applicationsDir, err := GetDesktopApplicationsDir()
	if err != nil {
		return oldDesktopFileName, err
	}
	oldDesktopFilePath := filepath.Join(applicationsDir, oldDesktopFileName)
	oldDesktopFileExists, err := PathExists(oldDesktopFilePath)
	if err != nil {
		return oldDesktopFileName, err
	}
	if !oldDesktopFileExists {
		return "", nil
	}

	desktopEntry, err := os.ReadFile(oldDesktopFilePath)
	if err != nil {
		return oldDesktopFileName, err
	}
	updatedDesktopEntry := updateNameAndExec(string(desktopEntry), newBinaryName, newBinaryPath)

	newDesktopFileName := DesktopFileName(newBinaryName)
	newDesktopFilePath := filepath.Join(applicationsDir, newDesktopFileName)
	if newDesktopFileName == oldDesktopFileName {
		if err := os.WriteFile(oldDesktopFilePath, []byte(updatedDesktopEntry), 0644); err != nil {
			return oldDesktopFileName, err
		}
		return newDesktopFileName, nil
	}

	newDesktopFileExists, err := PathExists(newDesktopFilePath)
	if err != nil {
		return oldDesktopFileName, err
	}
	if newDesktopFileExists {
		fmt.Printf("%v The desktop file %v already exists, skipping the desktop file rename\n", constants.YellowColor("WARNING:"), constants.GreenColor(newDesktopFilePath))
		return oldDesktopFileName, nil
	}

	if err := os.WriteFile(newDesktopFilePath, []byte(updatedDesktopEntry), 0644); err != nil {
		return oldDesktopFileName, err
	}
	if err := os.Remove(oldDesktopFilePath); err != nil {
		return oldDesktopFileName, err
	}
	updateDesktopDatabase(applicationsDir)

	return newDesktopFileName, nil
}

// DeleteDesktopFile deletes a tracked .desktop file. A missing .desktop file is not an error.
func DeleteDesktopFile(desktopFileName string) error {
	if desktopFileName == "" {
		return nil
	}

	applicationsDir, err := GetDesktopApplicationsDir()
	if err != nil {
		return err
	}
	desktopFilePath := filepath.Join(applicationsDir, desktopFileName)
	desktopFileExists, err := PathExists(desktopFilePath)
	if err != nil {
		return err
	}
	if !desktopFileExists {
		return nil
	}

	if err := os.Remove(desktopFilePath); err != nil {
		return err
	}
	updateDesktopDatabase(applicationsDir)

	return nil
}
