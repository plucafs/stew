package stew

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopFileName(t *testing.T) {
	desktopFileName := DesktopFileName("gh")
	expectedDesktopFileName := "stew-gh.desktop"
	if desktopFileName != expectedDesktopFileName {
		t.Errorf("expected desktop file name %v, got %v", expectedDesktopFileName, desktopFileName)
	}
}

func TestGetDesktopApplicationsDir(t *testing.T) {
	xdgDataHomePath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHomePath)

	applicationsDir, err := GetDesktopApplicationsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedApplicationsDir := filepath.Join(xdgDataHomePath, "applications")
	if applicationsDir != expectedApplicationsDir {
		t.Errorf("expected applications dir %v, got %v", expectedApplicationsDir, applicationsDir)
	}

	t.Setenv("XDG_DATA_HOME", "")
	applicationsDir, err = GetDesktopApplicationsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedApplicationsDir = filepath.Join(homeDir, ".local", "share", "applications")
	if applicationsDir != expectedApplicationsDir {
		t.Errorf("expected applications dir %v, got %v", expectedApplicationsDir, applicationsDir)
	}
}

func TestBuildDesktopEntry(t *testing.T) {
	desktopEntry := buildDesktopEntry("gh", "/home/user/.local/bin/gh", "GitHub's command line tool")

	expectedDesktopEntry := `[Desktop Entry]
Type=Application
Name=gh
Comment=GitHub's command line tool
Exec="/home/user/.local/bin/gh"
Icon=utilities-terminal
Terminal=false
Categories=Utility;
`
	if desktopEntry != expectedDesktopEntry {
		t.Errorf("expected desktop entry:\n%v\ngot:\n%v", expectedDesktopEntry, desktopEntry)
	}

	desktopEntryWithoutComment := buildDesktopEntry("gh", "/home/user/.local/bin/gh", "")
	if strings.Contains(desktopEntryWithoutComment, "Comment=") {
		t.Errorf("expected no Comment line, got:\n%v", desktopEntryWithoutComment)
	}

	desktopEntryWithMultilineComment := buildDesktopEntry("gh", "/home/user/.local/bin/gh", "line one\nline two")
	if !strings.Contains(desktopEntryWithMultilineComment, "Comment=line one line two") {
		t.Errorf("expected newlines in the comment to be replaced, got:\n%v", desktopEntryWithMultilineComment)
	}

	desktopEntryWithQuotedExec := buildDesktopEntry("gh", `/home/user/my "quoted" dir\gh`, "")
	if !strings.Contains(desktopEntryWithQuotedExec, `Exec="/home/user/my \"quoted\" dir\\gh"`) {
		t.Errorf("expected the Exec path to be quoted and escaped, got:\n%v", desktopEntryWithQuotedExec)
	}
}

func TestCreateDesktopFile(t *testing.T) {
	applicationsDir := t.TempDir()
	desktopFilePath := filepath.Join(applicationsDir, "stew-gh.desktop")

	err := CreateDesktopFile(desktopFilePath, "gh", "/home/user/.local/bin/gh", "GitHub's command line tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	desktopFileContents, err := os.ReadFile(desktopFilePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(desktopFileContents), "Name=gh") {
		t.Errorf("expected Name=gh in the desktop file, got:\n%v", string(desktopFileContents))
	}

	err = CreateDesktopFile(desktopFilePath, "gh", "/home/user/.local/bin/gh", "")
	if !os.IsExist(err) {
		t.Errorf("expected a file exists error when overwriting, got: %v", err)
	}
}

func TestUpdateNameAndExec(t *testing.T) {
	desktopEntry := `[Desktop Entry]
Type=Application
Name=gh
Exec="/home/user/.local/bin/gh"
Icon=custom-icon
Terminal=false
Categories=Utility;

[Desktop Action open]
Name=Open gh
Exec="/home/user/.local/bin/gh" --web
`

	updatedDesktopEntry := updateNameAndExec(desktopEntry, "gh2", "/home/user/.local/bin/gh2")

	expectedUpdatedDesktopEntry := `[Desktop Entry]
Type=Application
Name=gh2
Exec="/home/user/.local/bin/gh2"
Icon=custom-icon
Terminal=false
Categories=Utility;

[Desktop Action open]
Name=Open gh
Exec="/home/user/.local/bin/gh" --web
`
	if updatedDesktopEntry != expectedUpdatedDesktopEntry {
		t.Errorf("expected updated desktop entry:\n%v\ngot:\n%v", expectedUpdatedDesktopEntry, updatedDesktopEntry)
	}
	if !strings.Contains(updatedDesktopEntry, "Icon=custom-icon") {
		t.Errorf("expected a hand-edited icon value to be preserved, got:\n%v", updatedDesktopEntry)
	}
}

func TestRenameDesktopFile(t *testing.T) {
	xdgDataHomePath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHomePath)
	applicationsDir := filepath.Join(xdgDataHomePath, "applications")
	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldDesktopFilePath := filepath.Join(applicationsDir, "stew-gh.desktop")
	desktopEntry := `[Desktop Entry]
Type=Application
Name=gh
Exec="/home/user/.local/bin/gh"
Icon=custom-icon
Terminal=false
Categories=Utility;
`
	if err := os.WriteFile(oldDesktopFilePath, []byte(desktopEntry), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newDesktopFileName, err := RenameDesktopFile("stew-gh.desktop", "gh2", "/home/user/.local/bin/gh2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newDesktopFileName != "stew-gh2.desktop" {
		t.Errorf("expected stew-gh2.desktop, got %v", newDesktopFileName)
	}

	newDesktopFilePath := filepath.Join(applicationsDir, "stew-gh2.desktop")
	newDesktopFileContents, err := os.ReadFile(newDesktopFilePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(newDesktopFileContents), "Name=gh2") {
		t.Errorf("expected Name=gh2 in the renamed desktop file, got:\n%v", string(newDesktopFileContents))
	}
	if !strings.Contains(string(newDesktopFileContents), `Exec="/home/user/.local/bin/gh2"`) {
		t.Errorf("expected the Exec value to point at the renamed binary, got:\n%v", string(newDesktopFileContents))
	}
	if !strings.Contains(string(newDesktopFileContents), "Icon=custom-icon") {
		t.Errorf("expected a hand-edited icon value to be preserved, got:\n%v", string(newDesktopFileContents))
	}
	if _, err := os.Stat(oldDesktopFilePath); !os.IsNotExist(err) {
		t.Errorf("expected the old desktop file to be removed")
	}
}

func TestRenameDesktopFileCollision(t *testing.T) {
	xdgDataHomePath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHomePath)
	applicationsDir := filepath.Join(xdgDataHomePath, "applications")
	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldDesktopFilePath := filepath.Join(applicationsDir, "stew-gh.desktop")
	newDesktopFilePath := filepath.Join(applicationsDir, "stew-gh2.desktop")
	if err := os.WriteFile(oldDesktopFilePath, []byte("[Desktop Entry]\nName=gh\n"), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(newDesktopFilePath, []byte("[Desktop Entry]\nName=other package\n"), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newDesktopFileName, err := RenameDesktopFile("stew-gh.desktop", "gh2", "/home/user/.local/bin/gh2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newDesktopFileName != "stew-gh.desktop" {
		t.Errorf("expected the old desktop file name to be kept, got %v", newDesktopFileName)
	}

	unchangedDesktopFileContents, err := os.ReadFile(newDesktopFilePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(unchangedDesktopFileContents), "Name=other package") {
		t.Errorf("expected the colliding desktop file to be untouched, got:\n%v", string(unchangedDesktopFileContents))
	}
}

func TestRenameDesktopFileMissingFile(t *testing.T) {
	xdgDataHomePath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHomePath)
	applicationsDir := filepath.Join(xdgDataHomePath, "applications")
	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newDesktopFileName, err := RenameDesktopFile("stew-gh.desktop", "gh2", "/home/user/.local/bin/gh2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newDesktopFileName != "" {
		t.Errorf("expected an empty desktop file name for a missing file, got %v", newDesktopFileName)
	}

	newDesktopFileName, err = RenameDesktopFile("", "gh2", "/home/user/.local/bin/gh2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newDesktopFileName != "" {
		t.Errorf("expected an empty desktop file name for an empty input, got %v", newDesktopFileName)
	}
}

func TestDeleteDesktopFile(t *testing.T) {
	xdgDataHomePath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHomePath)
	applicationsDir := filepath.Join(xdgDataHomePath, "applications")
	if err := os.MkdirAll(applicationsDir, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	desktopFilePath := filepath.Join(applicationsDir, "stew-gh.desktop")
	if err := os.WriteFile(desktopFilePath, []byte("[Desktop Entry]\nName=gh\n"), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := DeleteDesktopFile("stew-gh.desktop"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(desktopFilePath); !os.IsNotExist(err) {
		t.Errorf("expected the desktop file to be deleted")
	}

	if err := DeleteDesktopFile("stew-gh.desktop"); err != nil {
		t.Errorf("expected deleting a missing desktop file to be a no-op, got: %v", err)
	}
	if err := DeleteDesktopFile(""); err != nil {
		t.Errorf("expected deleting an empty desktop file name to be a no-op, got: %v", err)
	}
}

func TestDesktopFileLockFileJSON(t *testing.T) {
	packageData := PackageData{
		Source:      "github",
		Owner:       "cli",
		Repo:        "cli",
		Tag:         "v2.4.0",
		Asset:       "gh_2.4.0_macOS_amd64.tar.gz",
		Binary:      "gh",
		URL:         "https://github.com/cli/cli/releases/download/v2.4.0/gh_2.4.0_macOS_amd64.tar.gz",
		BinaryHash:  "abc123",
		DesktopFile: "stew-gh.desktop",
	}

	packageDataJSON, err := json.Marshal(packageData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(packageDataJSON), `"desktopFile":"stew-gh.desktop"`) {
		t.Errorf("expected the desktopFile field to be serialized, got: %v", string(packageDataJSON))
	}

	packageData.DesktopFile = ""
	packageDataJSON, err = json.Marshal(packageData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(packageDataJSON), "desktopFile") {
		t.Errorf("expected an empty desktopFile field to be omitted, got: %v", string(packageDataJSON))
	}

	legacyPackageDataJSON := `{"source":"github","owner":"cli","repo":"cli","tag":"v2.4.0","asset":"gh_2.4.0_macOS_amd64.tar.gz","binary":"gh","url":"https://github.com/cli/cli/releases/download/v2.4.0/gh_2.4.0_macOS_amd64.tar.gz","binaryHash":"abc123"}`
	var legacyPackageData PackageData
	if err := json.Unmarshal([]byte(legacyPackageDataJSON), &legacyPackageData); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if legacyPackageData.DesktopFile != "" {
		t.Errorf("expected an empty desktopFile field for a legacy lockfile entry, got: %v", legacyPackageData.DesktopFile)
	}
}
