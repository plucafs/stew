package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marwanhawari/stew/constants"
	stew "github.com/marwanhawari/stew/lib"
)

// Rename is executed when you run `stew rename`
func Rename(cliInput string) {

	userOS, userArch, _, systemInfo, err := stew.Initialize()
	stew.CatchAndExit(err)

	err = stew.ValidateCLIInput(cliInput)
	stew.CatchAndExit(err)

	stewBinPath := systemInfo.StewBinPath
	stewLockFilePath := systemInfo.StewLockFilePath

	lockFile, err := stew.NewLockFile(stewLockFilePath, userOS, userArch)
	stew.CatchAndExit(err)

	if len(lockFile.Packages) == 0 {
		stew.CatchAndExit(stew.NoBinariesInstalledError{})
	}

	var binaryFound bool
	var renamedBinaryName string
	var indexInLockFile int
	var desktopFileName string
	for index, pkg := range lockFile.Packages {
		if pkg.Binary == cliInput {
			renamedBinaryName, err = stew.PromptRenameBinary(cliInput)
			stew.CatchAndExit(err)
			err = os.Rename(filepath.Join(stewBinPath, cliInput), filepath.Join(stewBinPath, renamedBinaryName))
			stew.CatchAndExit(err)

			lockFile.Packages[index].Binary = renamedBinaryName
			indexInLockFile = index
			desktopFileName = pkg.DesktopFile
			binaryFound = true
			break
		}
	}
	if !binaryFound {
		stew.CatchAndExit(stew.BinaryNotInstalledError{Binary: cliInput})
	}

	newDesktopFileName, err := stew.RenameDesktopFile(desktopFileName, renamedBinaryName, filepath.Join(stewBinPath, renamedBinaryName))
	if err != nil {
		fmt.Printf("%v Could not update the desktop file for %v: %v\n", constants.YellowColor("WARNING:"), constants.YellowColor(cliInput), err)
	} else {
		lockFile.Packages[indexInLockFile].DesktopFile = newDesktopFileName
	}

	err = stew.WriteLockFileJSON(lockFile, stewLockFilePath)
	stew.CatchAndExit(err)

	fmt.Printf("✨ Successfully renamed the %v binary to %v\n", constants.GreenColor(cliInput), constants.GreenColor(renamedBinaryName))
}
