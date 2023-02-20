package confs

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/moqsien/gvc/pkgs/utils"
)

/*
gvc related
*/
const GVCVersion = "v0.1.0"

var (
	GVCWorkDir          = filepath.Join(utils.GetHomeDir(), ".gvc/")
	GVCWebdavConfigPath = filepath.Join(GVCWorkDir, "webdav.yml")
	GVCBackupDir        = filepath.Join(GVCWorkDir, "backup")
	GVConfigPath        = filepath.Join(GVCBackupDir, "gvc-config.yml")
)

/*
hosts related
*/
const (
	HostFilePathForNix = "/etc/hosts"
	HostFilePathForWin = `C:\Windows\System32\drivers\etc\hosts`
)

var TempHostsFilePath = filepath.Join(GVCWorkDir, "/temp_hosts.txt")

func GetHostsFilePath() (r string) {
	if runtime.GOOS == "windows" {
		r = HostFilePathForWin
	}
	r = HostFilePathForNix
	return r
}

/*
vscode related
*/
var (
	CodeFileDir         string = filepath.Join(GVCWorkDir, "vscode_file")
	CodeTarFileDir      string = filepath.Join(CodeFileDir, "downloads")
	CodeUntarFile       string = filepath.Join(CodeFileDir, "vscode")
	CodeMacInstallDir   string = "/Applications/"
	CodeMacCmdBinaryDir string = filepath.Join(CodeMacInstallDir, "Visual Studio Code.app/Contents/Resources/app/bin")
	CodeWinCmdBinaryDir string = filepath.Join(CodeUntarFile, "bin")
	CodeWinShortcutPath string = filepath.Join(utils.GetHomeDir(), "Desktop/", "Visual Studio Code")
)

var (
	CodeEnvForUnix string = `# VSCode start
export PATH="%s:$PATH"
# VSCode end`
)

var (
	CodeUserSettingsFilePathForMac string = filepath.Join(utils.GetHomeDir(),
		"Library/Application Support/Code/User/settings.json")
	CodeKeybindingsFilePathForMac string = filepath.Join(utils.GetHomeDir(),
		"Library/Application Support/Code/User/keybindings.json")
	CodeUserSettingsFilePathForWin   string = ""
	CodeKeybindingsFilePathForWin    string = ""
	CodeUserSettingsFilePathForLinux string = ""
	CodeKeybindingsFilePathForLinux  string = ""
	CodeUserSettingsBackupPath              = filepath.Join(GVCBackupDir, "vscode-settings.json")
	CodeKeybindingsBackupPath               = filepath.Join(GVCBackupDir, "vscode-keybindings.json")
)

func GetCodeUserSettingsPath() string {
	switch runtime.GOOS {
	case "darwin":
		return CodeUserSettingsFilePathForMac
	case "linux":
		return CodeUserSettingsFilePathForLinux
	case "windows":
		return CodeUserSettingsFilePathForWin
	default:
		return ""
	}
}

func GetCodeKeybindingsPath() string {
	switch runtime.GOOS {
	case "darwin":
		return CodeKeybindingsFilePathForMac
	case "linux":
		return CodeKeybindingsFilePathForLinux
	case "windows":
		return CodeKeybindingsFilePathForWin
	default:
		return ""
	}
}

/*
go related
*/
var GoFilesDir = filepath.Join(GVCWorkDir, "go_files")

var (
	DefaultGoRoot    string = filepath.Join(GoFilesDir, "go")
	DefaultGoPath    string = filepath.Join(utils.GetHomeDir(), "data/projects/go")
	DefaultGoProxy   string = "https://goproxy.cn,direct"
	GoTarFilesPath   string = filepath.Join(GoFilesDir, "downloads")
	GoUnTarFilesPath string = filepath.Join(GoFilesDir, "versions")
)

var (
	GoUnixEnvsPattern string = `# Golang Start
export GOROOT="%s"
export GOPATH="%s"
export GOBIN="%s"
export GOPROXY="%s"
export PATH="%s"
# Golang End`
	GoUnixEnv string = fmt.Sprintf(GoUnixEnvsPattern,
		DefaultGoRoot,
		DefaultGoPath,
		filepath.Join(DefaultGoPath, "bin"),
		`%s`,
		`%s`)
)

// var (
// 	GoWinBatPattern string = `@echo off
// setx "GOROOT" "%s"
// setx "GOPATH" "%s"
// setx "GORIN" "%s"
// setx "GOPROXY" "%s"
// setx Path "%s"
// @echo on
// `
// 	GoWinBatPath string = filepath.Join(GoFilesDir, "genv.bat")
// 	GoWinEnv     string = fmt.Sprintf(GoWinBatPattern,
// 		DefaultGoRoot,
// 		DefaultGoPath,
// 		filepath.Join(DefaultGoPath, "bin"),
// 		`%s`,
// 		`%s`)
// )

/*
Neovim related.
*/
var (
	NVimFileDir        string = filepath.Join(GVCWorkDir, "nvim_files")
	NVimWinInitPath    string = filepath.Join(utils.GetHomeDir(), `\AppData\Local\nvim\init.vim`)
	NVimUnixInitPath   string = filepath.Join(utils.GetHomeDir(), ".config/nvim/init.vim")
	NVimInitBackupPath string = filepath.Join(GVCBackupDir, "nvim-init.vim")
)

func GetNVimInitPath() string {
	if runtime.GOOS == "windows" {
		return NVimWinInitPath
	}
	return NVimUnixInitPath
}

func GetNVimPlugDir() string {
	return filepath.Dir(GetNVimInitPath())
}

var (
	NVimUnixEnv string = `# nvim start
export PATH="%s:$PATH"
# nvim end`
)
