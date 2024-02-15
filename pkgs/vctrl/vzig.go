package vctrl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gogf/gf/v2/os/genv"
	"github.com/gvcgo/goutils/pkgs/gtea/gprint"
	"github.com/gvcgo/goutils/pkgs/request"
	"github.com/mholt/archiver/v3"
	config "github.com/gvcgo/gvc/pkgs/confs"
	"github.com/gvcgo/gvc/pkgs/utils"
)

var ZigOSArchMap = map[string]string{
	"windows-x86_64":  "windows_amd64",
	"windows-aarch64": "windows_arm64",
	"macos-x86_64":    "darwin_amd64",
	"macos-aarch64":   "darwin_arm64",
	"linux-x86_64":    "linux_amd64",
	"linux-aarch64":   "linux_arm64",
}

/*
https://github.com/ziglang/zig
https://ziglang.org/
https://github.com/zigtools/zls/wiki/Installation
*/
type Zig struct {
	Conf    *config.GVConfig
	env     *utils.EnvsHandler
	fetcher *request.Fetcher
	zigList map[string]string
}

func NewZig() (z *Zig) {
	z = &Zig{
		Conf:    config.New(),
		fetcher: request.NewFetcher(),
		env:     utils.NewEnvsHandler(),
		zigList: map[string]string{},
	}
	z.env.SetWinWorkDir(config.GVCDir)
	return
}

func (that *Zig) GetZigList() {
	if len(that.zigList) > 0 {
		return
	}
	that.fetcher.SetUrl(that.Conf.Zig.ZigDownloadUrl)
	that.fetcher.Timeout = time.Minute * 5
	if resp := that.fetcher.Get(); resp != nil {
		doc, err := goquery.NewDocumentFromReader(resp.RawBody())
		if err != nil || doc == nil {
			gprint.PrintError(fmt.Sprintf("Cannot parse html for %s", that.fetcher.Url))
			os.Exit(1)
		}
		// Latest stable version only.
		doc.Find("table").Eq(1).Find("a").Each(func(i int, s *goquery.Selection) {
			href := s.AttrOr("href", "")
			gh := NewGhDownloader()
			if href != "" && !strings.Contains(href, "minisig") {
				osInfo, archInfo := gh.ParseOSAndArchFromFileName(filepath.Base(href))
				if osInfo != "" && archInfo != "" {
					that.zigList[fmt.Sprintf("%s_%s", osInfo, archInfo)] = href
					return
				}
				if osInfo == utils.MacOS && archInfo == "" {
					that.zigList[fmt.Sprintf("%s_%s", osInfo, "amd64")] = href
					that.zigList[fmt.Sprintf("%s_%s", osInfo, "arm64")] = href
				} else if osInfo != "" && archInfo == "" {
					that.zigList[fmt.Sprintf("%s_%s", osInfo, "amd64")] = href
				}
				// for k, v := range ZigOSArchMap {
				// 	if strings.Contains(href, k) && !strings.Contains(href, "minisig") {
				// 		that.zigList[v] = href
				// 	}
				// }
			}
		})
	}
}

func (that *Zig) download(force bool) (fPath string) {
	that.GetZigList()
	dUrl := that.zigList[fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)]
	if dUrl == "" {
		gprint.PrintError("Cannot find download url.")
		return
	}
	gprint.PrintInfo("download from: %s", dUrl)
	that.fetcher.SetUrl(dUrl)
	that.fetcher.Timeout = time.Minute * 30
	that.fetcher.SetThreadNum(3)
	fName := "zig.tar.xz"
	if strings.HasSuffix(dUrl, ".zip") {
		fName = "zig.zip"
	}
	fp := filepath.Join(config.ZigFilesDir, fName)
	if force {
		os.RemoveAll(fp)
	}
	if ok, _ := utils.PathIsExist(fp); !ok || force {
		if size := that.fetcher.GetAndSaveFile(fp); size > 0 {
			return fp
		} else {
			os.RemoveAll(fp)
		}
	} else if ok && !force {
		return fp
	}
	return
}

func (that *Zig) Install(force bool) {
	fPath := that.download(force)
	if fPath == "" {
		gprint.PrintError("download failed.")
		return
	}
	if ok, _ := utils.PathIsExist(config.ZigRootDir); ok && !force {
		gprint.PrintInfo("Zig is already installed.")
		return
	} else {
		os.RemoveAll(config.ZigRootDir)
	}
	if err := archiver.Unarchive(fPath, config.ZigFilesDir); err != nil {
		os.RemoveAll(config.ZigRootDir)
		os.RemoveAll(fPath)
		gprint.PrintError(fmt.Sprintf("Unarchive failed: %+v", err))
		return
	}
	that.renameZigDir()
	if ok, _ := utils.PathIsExist(config.ZigRootDir); ok {
		that.CheckAndInitEnv()
	}
	gprint.PrintSuccess("Installation succeeded.")
}

func (that *Zig) renameZigDir() {
	itemList, _ := os.ReadDir(config.ZigFilesDir)
	for _, item := range itemList {
		if item.IsDir() && strings.Contains(item.Name(), "zig-") {
			untarredDir := filepath.Join(config.ZigFilesDir, item.Name())
			os.Rename(untarredDir, config.ZigRootDir)
		}
	}
}

func (that *Zig) CheckAndInitEnv() {
	zlsBinDir := filepath.Join(config.ZlsRootDir, "bin")
	exist, _ := utils.PathIsExist(zlsBinDir)
	if runtime.GOOS != utils.Windows {
		zigEnv := fmt.Sprintf(utils.ZigEnv, config.ZigRootDir)
		if exist {
			zlsEnv := fmt.Sprintf(utils.ZigEnv, zlsBinDir)
			zigEnv = fmt.Sprintf("%s\n%s", zigEnv, zlsEnv)
		}
		that.env.UpdateSub(utils.SUB_ZIG, zigEnv)
	} else {
		that.env.SetEnvForWin(map[string]string{
			"PATH": config.ZigRootDir,
		})
		if exist {
			that.env.SetEnvForWin(map[string]string{
				"PATH": filepath.Join(config.ZlsRootDir, "bin"),
			})
		}
	}
}

func (that *Zig) downloadZls(force bool) (fPath string) {
	gh := NewGhDownloader()
	uList := gh.ParseReleasesForGithubProject(that.Conf.Zig.ZlsDownloadUrl)
	dUrl := uList[fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)]
	if dUrl == "" {
		gprint.PrintError("Cannot find download url.")
		return
	}
	gprint.PrintInfo("download from: %s", dUrl)
	that.fetcher.SetUrl(that.Conf.GVCProxy.WrapUrl(dUrl))
	that.fetcher.Timeout = time.Minute * 30
	that.fetcher.SetThreadNum(3)
	fName := "zls.tar.gz"
	if strings.HasSuffix(dUrl, ".zip") {
		fName = "zls.zip"
	}
	fp := filepath.Join(config.ZigFilesDir, fName)
	if force {
		os.RemoveAll(fp)
	}
	if ok, _ := utils.PathIsExist(fp); !ok || force {
		if size := that.fetcher.GetAndSaveFile(fp); size > 0 {
			return fp
		} else {
			os.RemoveAll(fp)
		}
	} else if ok && !force {
		return fp
	}
	return
}

func (that *Zig) renameZlsDir() {
	itemList, _ := os.ReadDir(config.ZigFilesDir)
	for _, item := range itemList {
		if item.IsDir() && strings.Contains(item.Name(), "zls-") {
			untarredDir := filepath.Join(config.ZigFilesDir, item.Name())
			os.Rename(untarredDir, config.ZlsRootDir)
		}
	}
	binDirPath := filepath.Join(config.ZlsRootDir, "bin")
	binaryPath := filepath.Join(binDirPath, "zls")
	if runtime.GOOS == utils.Windows {
		binaryPath = filepath.Join(binDirPath, "zls.exe")
	}
	os.Chmod(binaryPath, 0777)
}

func (that *Zig) InstalZls(force bool) {
	fPath := that.downloadZls(force)
	if fPath == "" {
		gprint.PrintError("download zls failed.")
		return
	}
	if ok, _ := utils.PathIsExist(config.ZlsRootDir); ok && !force {
		gprint.PrintInfo("zls is already installed.")
		return
	} else {
		os.RemoveAll(config.ZlsRootDir)
	}

	if err := archiver.Unarchive(fPath, config.ZlsRootDir); err != nil {
		os.RemoveAll(config.ZlsRootDir)
		os.RemoveAll(fPath)
		gprint.PrintError(fmt.Sprintf("Unarchive failed: %+v", err))
		return
	}
	that.renameZlsDir()
	if ok, _ := utils.PathIsExist(config.ZlsRootDir); ok {
		that.CheckAndInitEnv()
	}

	gprint.PrintSuccess("Zls installation succeeded.")
	// install zls extension for vscode.
	// 	AugusteRame.zls-vscode
	cmd := exec.Command("code", "--install-extension", "AugusteRame.zls-vscode")
	cmd.Env = genv.All()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Run()
}
