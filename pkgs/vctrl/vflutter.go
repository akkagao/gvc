package vctrl

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/mholt/archiver/v3"
	"github.com/moqsien/goutils/pkgs/gtea/gprint"
	"github.com/moqsien/goutils/pkgs/gtea/selector"
	"github.com/moqsien/goutils/pkgs/request"
	config "github.com/moqsien/gvc/pkgs/confs"
	"github.com/moqsien/gvc/pkgs/utils"
	"github.com/moqsien/gvc/pkgs/utils/sorts"
)

// only in china mianland
type FlutterPackage struct {
	Url         string
	FileName    string
	OS          string
	Arch        string
	DartVersion string
	Checksum    string
}

type FlutterVersion struct {
	Versions    map[string][]*FlutterPackage
	Json        *gjson.Json
	Conf        *config.GVConfig
	fetcher     *request.Fetcher
	env         *utils.EnvsHandler
	baseUrl     string
	flutterConf map[string]string
}

func NewFlutterVersion() (fv *FlutterVersion) {
	fv = &FlutterVersion{
		Versions:    make(map[string][]*FlutterPackage, 500),
		Conf:        config.New(),
		fetcher:     request.NewFetcher(),
		env:         utils.NewEnvsHandler(),
		flutterConf: map[string]string{},
	}
	fv.initeDirs()
	fv.env.SetWinWorkDir(config.GVCWorkDir)
	return
}

func (that *FlutterVersion) initeDirs() {
	utils.MakeDirs(
		config.FlutterRootDir,
		config.FlutterTarFilePath,
		config.FlutterUntarFilePath,
		config.FlutterAndroidToolDownloads,
	)
}

func (that *FlutterVersion) ChooseSource() {
	if that.flutterConf == nil || len(that.flutterConf) == 0 {
		itemList := selector.NewItemList()
		itemList.Add("from flutter-io.cn", that.Conf.Flutter.DefaultURLs)
		itemList.Add("from googleapis.com", that.Conf.Flutter.OfficialURLs)
		sel := selector.NewSelector(
			itemList,
			selector.WithTitle("Choose download resource:"),
			selector.WithEnbleInfinite(true),
			selector.WidthEnableMulti(false),
			selector.WithHeight(10),
			selector.WithWidth(40),
		)
		sel.Run()

		value := sel.Value()[0]
		that.flutterConf = value.(map[string]string)
	}
}

func (that *FlutterVersion) getJson() {
	that.ChooseSource()
	fUrl := that.flutterConf[runtime.GOOS]
	if !utils.VerifyUrls(fUrl) {
		return
	}

	that.fetcher.Url = fUrl
	if resp := that.fetcher.Get(); resp != nil {
		content, _ := io.ReadAll(resp.RawBody())
		that.Json = gjson.New(content)
	}
	if that.Json != nil {
		that.baseUrl = that.Json.GetString("base_url")
	}
}

func (that *FlutterVersion) GetFileSuffix(fName string) string {
	for _, k := range AllowedSuffixes {
		if strings.HasSuffix(fName, k) {
			return k
		}
	}
	return ""
}

func (that *FlutterVersion) GetVersions() {
	if that.Json == nil {
		that.getJson()
	}
	if that.Json != nil {
		rList := that.Json.GetArray("releases")
		for _, release := range rList {
			j := gjson.New(release)
			rChannel := j.GetString("channel")
			version := j.GetString("version")
			if rChannel != "stable" || version == "" || strings.Contains(version, "hotfix") {
				continue
			}

			p := &FlutterPackage{}
			p.Url = j.GetString("archive")
			p.Arch = utils.ParseArch(j.GetString("dart_sdk_arch"))
			if p.Url == "" || p.Arch == "" {
				continue
			}
			p.OS = runtime.GOOS
			p.DartVersion = j.GetString("dart_sdk_version")
			p.Checksum = j.GetString("sha256")
			p.FileName = fmt.Sprintf("flutter-%s-%s-%s%s",
				version, p.OS, p.Arch, that.GetFileSuffix(p.Url))
			if len(that.Versions[version]) == 0 {
				that.Versions[version] = []*FlutterPackage{p}
			} else {
				that.Versions[version] = append(that.Versions[version], p)
			}
		}
	}
}

func (that *FlutterVersion) ShowVersions() {
	// that.ChooseSource()
	if len(that.Versions) == 0 {
		that.GetVersions()
	}
	vList := []string{}
	for k := range that.Versions {
		vList = append(vList, k)
	}
	res := sorts.SortGoVersion(vList)
	fc := gprint.NewFadeColors(res)
	fc.Println()
}

func (that *FlutterVersion) findPackage(version string) *FlutterPackage {
	for _, pk := range that.Versions[version] {
		if pk.Arch == runtime.GOARCH && pk.OS == runtime.GOOS {
			return pk
		}
	}
	return nil
}

func (that *FlutterVersion) download(version string) (r string) {
	if len(that.Versions) == 0 || that.baseUrl == "" {
		that.GetVersions()
	}

	if p := that.findPackage(version); p != nil {
		that.fetcher.Url, _ = url.JoinPath(that.baseUrl, p.Url)
		if !utils.VerifyUrls(that.fetcher.Url) {
			return
		}
		that.fetcher.Timeout = 100 * time.Minute
		// that.fetcher.SetThreadNum(2)
		fpath := filepath.Join(config.FlutterTarFilePath, p.FileName)
		if size := that.fetcher.GetAndSaveFile(fpath); size > 0 {
			if p.Checksum != "" {
				if ok := utils.CheckFile(fpath, "sha256", p.Checksum); ok {
					return fpath
				} else {
					os.RemoveAll(fpath)
				}
			} else {
				return fpath
			}
		} else {
			os.RemoveAll(fpath)
		}
	} else {
		gprint.PrintError(fmt.Sprintf("Invalid Flutter version: %s", version))
	}
	return
}

func (that *FlutterVersion) CheckAndInitEnv() {
	that.ChooseSource()
	if runtime.GOOS != utils.Windows {
		flutterEnv := fmt.Sprintf(utils.FlutterEnv,
			config.FlutterRootDir,
			that.flutterConf["hosted_url"],
			that.flutterConf["storage_base_url"],
			that.flutterConf["git_url"])
		that.env.UpdateSub(utils.SUB_FLUTTER, flutterEnv)
	} else {
		envList := map[string]string{
			"PUB_HOSTED_URL":           that.flutterConf["hosted_url"],
			"FLUTTER_STORAGE_BASE_URL": that.flutterConf["storage_base_url"],
			"FLUTTER_GIT_URL":          that.flutterConf["git_url"],
			"PATH":                     filepath.Join(config.FlutterRootDir, "bin"),
		}
		that.env.SetEnvForWin(envList)
	}
}

func (that *FlutterVersion) FixForFlutter() {
	// git remote set-url origin https://mirrors.tuna.tsinghua.edu.cn/git/flutter-sdk.git
	cmd := exec.Command("git", "remote", "set-url", "origin", that.flutterConf["git_url"])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = config.FlutterRootDir
	if err := cmd.Run(); err != nil {
		gprint.PrintError("%+v", err)
	}
	str := `'Upstream repository https://github.com/flutter/flutter.git is not the same as FLUTTER_GIT_URL'`
	fPath := filepath.Join(config.FlutterRootDir, "packages", "flutter_tools", "test", "general.shard", "flutter_validator_test.dart")
	if content, err := os.ReadFile(fPath); err != nil {
		newContentStr := strings.ReplaceAll(string(content), str, fmt.Sprintf("FLUTTER_GIT_URL :%s", that.flutterConf["git_url"]))
		if newContentStr != "" {
			os.WriteFile(fPath, []byte(newContentStr), os.ModePerm)
		}
	}
}

func (that *FlutterVersion) UseVersion(version string) {
	current := that.getCurrent()
	if version == current {
		gprint.PrintSuccess(fmt.Sprintf("Use %s succeeded!", version))
		return
	}
	if ok, _ := utils.PathIsExist(config.FlutterRootDir); ok {
		os.RemoveAll(config.FlutterRootDir)
	}

	untarfile := config.FlutterFilesDir
	if tarfile := that.download(version); tarfile != "" {
		if err := archiver.Unarchive(tarfile, untarfile); err != nil {
			os.RemoveAll(untarfile)
			gprint.PrintError(fmt.Sprintf("Unarchive failed: %+v", err))
			return
		}
	}

	if ok, _ := utils.PathIsExist(config.FlutterRootDir); ok {
		if !that.env.DoesEnvExist(utils.SUB_FLUTTER) {
			that.CheckAndInitEnv()
		}
		if strings.Contains(that.flutterConf["hosted_url"], ".cn") {
			that.FixForFlutter()
		}
		gprint.PrintSuccess(fmt.Sprintf("Use %s succeeded!", version))
	} else {
		gprint.PrintError(fmt.Sprintf("Use %s failed!", version))
	}
}

func (that *FlutterVersion) getCurrent() string {
	content, _ := os.ReadFile(filepath.Join(config.FlutterRootDir, "version"))
	return strings.TrimSpace(string(content))
}

func (that *FlutterVersion) ShowInstalled() {
	current := that.getCurrent()
	dList, _ := os.ReadDir(config.FlutterTarFilePath)
	reg := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	for _, d := range dList {

		if !d.IsDir() {
			zipName := d.Name()
			result := reg.FindAll([]byte(zipName), -1)
			var versionName string
			if len(result) == 1 {
				versionName = string(result[0])
			}
			if versionName == "" {
				continue
			}
			switch versionName {
			case current:
				gprint.Yellow("%s <Current>", versionName)
			default:
				gprint.Cyan(versionName)
			}
		}
	}
}

func (that *FlutterVersion) removeTarFile(version string) {
	fName := fmt.Sprintf("flutter-%s-%s-%s", version, runtime.GOOS, runtime.GOARCH)
	dList, _ := os.ReadDir(config.FlutterTarFilePath)
	for _, d := range dList {
		if !d.IsDir() && strings.Contains(d.Name(), fName) {
			os.RemoveAll(filepath.Join(config.FlutterTarFilePath, d.Name()))
		}
	}
}

func (that *FlutterVersion) RemoveVersion(version string) {
	current := that.getCurrent()
	if version == current {
		return
	}
	dList, _ := os.ReadDir(config.FlutterUntarFilePath)
	for _, d := range dList {
		if d.IsDir() && d.Name() == version {
			os.RemoveAll(filepath.Join(config.FlutterUntarFilePath, d.Name()))
			that.removeTarFile(version)
		}
	}
}

func (that *FlutterVersion) RemoveUnused() {
	current := that.getCurrent()
	dList, _ := os.ReadDir(config.FlutterUntarFilePath)
	for _, d := range dList {
		if d.IsDir() && d.Name() != current {
			os.RemoveAll(filepath.Join(config.FlutterUntarFilePath, d.Name()))
			that.removeTarFile(d.Name())
		}
	}
}

/*
Install Android SDK for Flutter & VSCode
*/

func (that *FlutterVersion) GetAndroidSDKInfo() (androidSDKs map[string]string) {
	androidSDKs = map[string]string{}
	itemList := selector.NewItemList()
	itemList.Add("from developer.android.google.cn", that.Conf.Flutter.AndroidCN)
	itemList.Add("from developer.android.com", that.Conf.Flutter.Android)
	sel := selector.NewSelector(
		itemList,
		selector.WidthEnableMulti(false),
		selector.WithEnbleInfinite(true),
		selector.WithTitle("Choose a download resource:"),
	)
	sel.Run()
	val := sel.Value()[0]
	if aUrl := val.(string); aUrl != "" {
		dUrl := that.Conf.Flutter.AndroidCMDTooolsUrl
		if strings.Contains(aUrl, ".cn") {
			dUrl = that.Conf.Flutter.AndroidCMDToolsUrlCN
		}
		that.fetcher.SetUrl(aUrl)
		that.fetcher.Timeout = 3 * time.Minute
		content, _ := that.fetcher.GetString()
		if doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(content)); err == nil {
			doc.Find("table.download").Find("button").Each(func(i int, s *goquery.Selection) {
				text := s.Text()
				if !strings.Contains(text, "commandlinetools") {
					return
				}
				sUrl := dUrl + strings.TrimSpace(text)
				if strings.Contains(sUrl, "win") {
					androidSDKs[utils.Windows] = sUrl
				} else if strings.Contains(sUrl, "mac") {
					androidSDKs[utils.MacOS] = sUrl
				} else if strings.Contains(sUrl, "linux") {
					androidSDKs[utils.Linux] = sUrl
				}
			})
		} else {
			gprint.PrintError("%+v", err)
		}
	}
	return androidSDKs
}

func (that *FlutterVersion) InstallAndroidTool() {
	infoList := that.GetAndroidSDKInfo()
	if len(infoList) == 0 {
		return
	}
	dUrl := infoList[runtime.GOOS]
	if dUrl == "" {
		return
	}
	that.fetcher.SetUrl(dUrl)
	that.fetcher.SetThreadNum(2)
	that.fetcher.Timeout = time.Minute * 20
	untarDirPath := filepath.Join(config.FlutterFilesDir, "cmdline-tools")
	if ok, _ := utils.PathIsExist(untarDirPath); ok {
		os.RemoveAll(untarDirPath)
	}
	fPath := filepath.Join(config.FlutterAndroidToolDownloads, "android-cmdline-tools.zip")
	if size := that.fetcher.GetAndSaveFile(fPath, true); size > 500 {
		if err := archiver.Unarchive(fPath, config.FlutterFilesDir); err != nil {
			os.RemoveAll(fPath)
			gprint.PrintError("unarchive file failed: %+v", err)
		}
	}
	if ok, _ := utils.PathIsExist(untarDirPath); ok {
		that.SetEnvForAndroidTools(untarDirPath)
	}
}

func (that *FlutterVersion) SetEnvForAndroidTools(untarDirPath string) {
	if runtime.GOOS != utils.Windows {
		androidToolEnv := fmt.Sprintf(utils.AndroidEnv, filepath.Join(untarDirPath, "bin"))
		that.env.UpdateSub(utils.SUB_ANDROID, androidToolEnv)
	} else {
		envList := map[string]string{
			"PATH": filepath.Join(untarDirPath, "bin"),
		}
		that.env.SetEnvForWin(envList)
	}
}

// TODO: setup for AVD
