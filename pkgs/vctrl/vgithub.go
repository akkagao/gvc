package vctrl

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	lazyapp "github.com/jesseduffield/lazygit/pkg/app"
	"github.com/moqsien/goutils/pkgs/ggit"
	"github.com/moqsien/goutils/pkgs/gtea/gprint"
	"github.com/moqsien/goutils/pkgs/gtea/selector"
	"github.com/moqsien/goutils/pkgs/request"
	config "github.com/moqsien/gvc/pkgs/confs"
	"github.com/moqsien/gvc/pkgs/utils"
)

const (
	DefaultProxyFileName string = ".default_proxy.conf"
)

type GhDownloader struct {
	Conf     *config.GVConfig
	path     string
	fetcher  *request.Fetcher
	releases map[string]string
	git      *ggit.Git
	env      *utils.EnvsHandler
}

func NewGhDownloader() (gd *GhDownloader) {
	gd = &GhDownloader{
		path:     filepath.Join(utils.GetHomeDir(), "Downloads"),
		fetcher:  request.NewFetcher(),
		Conf:     config.New(),
		releases: make(map[string]string),
		git:      ggit.NewGit(),
		env:      utils.NewEnvsHandler(),
	}
	return
}

func (that *GhDownloader) findFileName(dUrl string) (name string) {
	if strings.Contains(dUrl, "/archive") {
		sList := strings.Split(dUrl, "github.com/")
		if len(sList) == 2 {
			s := sList[1]
			sList = strings.Split(s, "/")
			if len(sList) >= 2 {
				return fmt.Sprintf("%s_code.zip", sList[1])
			}
		}
		return "source_code.zip"
	} else {
		sList := strings.Split(dUrl, "/")
		return fmt.Sprintf("%s_code.zip", sList[len(sList)-1])
	}
}

func (that *GhDownloader) downloadArchive(githubProjectUrl string) {
	// example: https://github.com/moqsien/gvc/archive/refs/heads/main.zip
	mainZipUrl := githubProjectUrl + "/archive/refs/heads/main.zip"
	fPath := filepath.Join(that.path, that.findFileName(mainZipUrl))
	that.fetcher.SetUrl(that.Conf.Github.DownProxy + mainZipUrl)
	that.fetcher.Timeout = 30 * time.Minute
	gprint.PrintInfo(fmt.Sprintf("[>>>] %s", mainZipUrl))
	if size := that.fetcher.GetFile(fPath, true); size <= 99 {
		masterZipUrl := githubProjectUrl + "/archive/refs/heads/master.zip"
		fPath = filepath.Join(that.path, that.findFileName(masterZipUrl))
		that.fetcher.SetUrl(that.Conf.Github.DownProxy + masterZipUrl)
		that.fetcher.Timeout = 30 * time.Minute
		that.fetcher.GetFile(fPath, true)

	}
	gprint.PrintSuccess(fPath)
}

func (that *GhDownloader) getCurrentTag(githubProjectUrl string) (tag string) {
	// example: https://github.com/moqsien/gvc/releases/latest
	dUrl := githubProjectUrl + "/releases/latest"
	client := resty.New()
	client.SetTimeout(time.Minute * 3)
	if resp, err := client.R().SetDoNotParseResponse(true).Head(that.Conf.Github.DownProxy + dUrl); err == nil {
		_url := resp.RawResponse.Request.URL.String()
		sList := strings.Split(_url, "/")
		return sList[len(sList)-1]
	}
	gprint.PrintInfo("Latest tag: %s", tag)
	return
}

func (that *GhDownloader) downloadBinary(githubProjectUrl string) {
	// example: https://github.com/moqsien/gvc/releases/expanded_assets/v1.3.1
	if tag := that.getCurrentTag(githubProjectUrl); tag != "" {
		that.fetcher.Url = that.Conf.Github.DownProxy + githubProjectUrl + fmt.Sprintf("/releases/expanded_assets/%s", tag)
		that.fetcher.Timeout = time.Minute * 3
		if resp := that.fetcher.Get(); resp != nil {
			if doc, err := goquery.NewDocumentFromReader(resp.RawBody()); err == nil && doc != nil {
				doc.Find("ul").Find("a").Each(func(i int, s *goquery.Selection) {
					if _url := s.AttrOr("href", ""); _url != "" {
						if filename := s.Find("span").First().Text(); filename != "" && !strings.Contains(filename, "Source code") {
							that.releases[filename], _ = url.JoinPath("https://github.com", _url)
						}
					}
				})
			}
		}
		if len(that.releases) > 0 {

			itemList := selector.NewItemList()
			for opt := range that.releases {
				itemList.Add(opt, opt)
			}
			sel := selector.NewSelector(
				itemList,
				selector.WithTitle("Choose a file to download: "),
				selector.WithEnbleInfinite(true),
				selector.WidthEnableMulti(false),
				selector.WithWidth(40),
				selector.WithHeight(10),
			)
			sel.Run()
			value := sel.Value()[0]
			selected := value.(string)
			dUrl := that.releases[selected]
			if dUrl != "" {
				gprint.PrintInfo("[Download] %s", dUrl)
				that.fetcher.SetUrl(that.Conf.Github.DownProxy + dUrl)
				that.fetcher.SetThreadNum(4)
				that.fetcher.Timeout = 30 * time.Minute
				fPath := filepath.Join(that.path, selected)
				if size := that.fetcher.GetAndSaveFile(fPath, true); size > 0 {
					gprint.PrintSuccess(fPath)
				}
			}
		}
	}
}

func (that *GhDownloader) Download(githubProjectUrl string, getSourceCode bool) {
	// example: https://github.com/moqsien/gvc
	if !strings.Contains(githubProjectUrl, "github.com/") {
		return
	}
	githubProjectUrl = strings.Split(githubProjectUrl, "/archive")[0]
	githubProjectUrl = strings.Split(githubProjectUrl, "/releases")[0]
	githubProjectUrl = strings.TrimRight(githubProjectUrl, "/")
	if getSourceCode {
		that.downloadArchive(githubProjectUrl)
	} else {
		that.downloadBinary(githubProjectUrl)
	}
}

func (that *GhDownloader) OpenByBrowser(chosen int) {
	urlList := that.Conf.Github.AccelUrls
	if len(urlList) == 0 {
		gprint.PrintError("No github download acceleration available.")
		return
	}
	var gUrl string
	if chosen >= len(urlList) {
		gUrl = urlList[0]
	} else {
		gUrl = urlList[chosen]
	}
	if gUrl != "" {
		var cmd *exec.Cmd
		if runtime.GOOS == utils.MacOS {
			cmd = exec.Command("open", gUrl)
		} else if runtime.GOOS == utils.Linux {
			cmd = exec.Command("x-www-browser", gUrl)
		} else if runtime.GOOS == utils.Windows {
			cmd = exec.Command("cmd", "/c", "start", gUrl)
		} else {
			return
		}
		if err := cmd.Run(); err != nil {
			gprint.PrintError(fmt.Sprintf("Execution failed: %+v", err))
		}
	}
}

func (that *GhDownloader) SaveDefaultProxy(proxyUrl string) {
	filePath := filepath.Join(config.GVCDir, DefaultProxyFileName)
	if proxyUrl == "" {
		proxyUrl = "http://127.0.0.1:2023"
	}
	if err := os.WriteFile(filePath, []byte(proxyUrl), os.ModePerm); err == nil {
		gprint.PrintInfo("default proxy for github has been saved in %s", filePath)
	}
}

func (that *GhDownloader) ReadDefaultProxy() string {
	filePath := filepath.Join(config.GVCDir, DefaultProxyFileName)
	r, _ := os.ReadFile(filePath)
	if len(r) == 0 {
		return "http://127.0.0.1:2023"
	}
	return string(r)
}

func (that *GhDownloader) Clone(projectUrl, proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if _, err := that.git.CloneBySSH(projectUrl); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) Pull(proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if err := that.git.PullBySSH(); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) Push(proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if err := that.git.PushBySSH(); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) CommitAndPush(commitMsg, proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if err := that.git.CommitAndPush(commitMsg); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) AddTagAndPush(tag, proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if err := that.git.AddTagAndPushToRemote(tag); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) DelTagAndPush(tag, proxyUrl string) {
	that.git.SetProxyUrl(proxyUrl)
	if err := that.git.DeleteTagAndPushToRemote(tag); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) ShowLatestTag() {
	if err := that.git.ShowLatestTag(); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) downloadGitForWindows() {
	if runtime.GOOS != utils.Windows {
		return
	}
	gUrl := that.Conf.Github.WinGitUrls[runtime.GOARCH]
	if gUrl == "" {
		return
	}
	if ok, _ := utils.PathIsExist(config.GitWindowsInstallationDir); !ok {
		os.MkdirAll(config.GitWindowsInstallationDir, os.ModePerm)
	}
	fPath := filepath.Join(config.GitFileDir, "git.7z")
	gUrl = that.Conf.GVCProxy.WrapUrl(gUrl)
	that.fetcher.SetUrl(gUrl)
	that.fetcher.SetThreadNum(2)
	that.fetcher.Timeout = 10 * time.Minute
	if err := that.fetcher.DownloadAndDecompress(fPath, config.GitWindowsInstallationDir, true); err != nil {
		gprint.PrintError("%+v", err)
	}
}

func (that *GhDownloader) InstallGitForWindows() {
	if runtime.GOOS != utils.Windows {
		return
	}
	os.RemoveAll(config.GitWindowsInstallationDir)
	that.downloadGitForWindows()

	binPath := filepath.Join(config.GitWindowsInstallationDir, "bin")
	usrBinPath := filepath.Join(config.GitWindowsInstallationDir, "usr", "bin")
	envarList := map[string]string{
		"PATH": fmt.Sprintf("%s;%s", binPath, usrBinPath),
	}
	that.env.SetEnvForWin(envarList)
}

func (that *GhDownloader) SetReverseProxyForDownload(pUrl string) {
	if pUrl == "" {
		return
	}
	if !strings.HasSuffix(pUrl, "/") {
		pUrl += "/"
	}
	that.Conf.Reload()
	that.Conf.GVCProxy.ReverseProxyUrl = pUrl
	that.Conf.Restore()
}

// set a proxy for git ssh
func (that *GhDownloader) SetProxyForGitSSH(pURI string) {
	if pURI == "" {
		pURI = that.ReadDefaultProxy()
	}
	if !strings.Contains(pURI, "://") {
		gprint.PrintError("No legal proxy is specified.")
		return
	}

	if u, err := url.Parse(pURI); err == nil {
		homeDir, _ := os.UserHomeDir()
		dotSSHPath := filepath.Join(homeDir, ".ssh")
		idRSAPath := filepath.Join(dotSSHPath, "id_rsa")
		if ok, _ := utils.PathIsExist(idRSAPath); !ok {
			gprint.PrintError("Cannot find ~/.ssh/id_rsa.")
			return
		}
		uStr := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
		switch runtime.GOOS {
		case utils.Windows:
			pxyCmd := fmt.Sprintf(
				config.GitSSHProxyCommandWin,
				uStr,
				`%h`,
				`%p`,
			)
			content := fmt.Sprintf(
				config.GitSSHConfigStr,
				idRSAPath,
				pxyCmd,
				idRSAPath,
				pxyCmd,
			)
			that.setProxyForGitSSH(dotSSHPath, idRSAPath, content)
		case utils.Linux, utils.MacOS:
			pxyCmd := fmt.Sprintf(
				config.GitSSHProxyCommandNix,
				uStr,
				`%h`,
				`%p`,
			)
			content := fmt.Sprintf(
				config.GitSSHConfigStr,
				idRSAPath,
				pxyCmd,
				idRSAPath,
				pxyCmd,
			)
			that.setProxyForGitSSH(dotSSHPath, idRSAPath, content)
		default:
			gprint.PrintError("Unsupported OS.")
		}
	}
}

func (that *GhDownloader) setProxyForGitSSH(dotSSHPath, idRSAPath, content string) {
	confPath := filepath.Join(dotSSHPath, "config")
	if ok, _ := utils.PathIsExist(confPath); !ok {
		os.WriteFile(confPath, []byte(content), 0666)
	} else {
		oldContentByte, _ := os.ReadFile(confPath)
		oldContent := string(oldContentByte)
		if !strings.Contains(oldContent, "ProxyCommand") && len(oldContent) > 0 {
			os.WriteFile(confPath, []byte(oldContent+"\n"+content), 0666)
		} else {
			os.WriteFile(confPath, []byte(content), 0666)
		}
	}
}

func (that *GhDownloader) ToggleProxyForGitSSH() {
	homeDir, _ := os.UserHomeDir()
	confPath := filepath.Join(homeDir, ".ssh", "config")
	backupConfPath := filepath.Join(homeDir, ".ssh", "config.bak")

	ok1, _ := utils.PathIsExist(confPath)
	ok2, _ := utils.PathIsExist(backupConfPath)

	if !ok1 && !ok2 {
		gprint.PrintError("No proxy available.")
	} else if ok1 && !ok2 {
		if err := os.Rename(confPath, backupConfPath); err == nil {
			gprint.PrintInfo("Proxy disabled.")
		}
	} else if !ok1 && ok2 {
		if err := os.Rename(backupConfPath, confPath); err == nil {
			gprint.PrintSuccess("Proxy enabled.")
		}
	} else {
		os.RemoveAll(backupConfPath)
		if err := os.Rename(confPath, backupConfPath); err == nil {
			gprint.PrintInfo("Proxy disabled.")
		}
	}
}

// lazygit with a proxy
func (that *GhDownloader) LazyGit(enableProxy bool, args ...string) {
	if enableProxy {
		pxyURI := that.ReadDefaultProxy()
		if pxyURI == "" {
			gprint.PrintError(`No proxy available. Please use command "git proxy <your_proxy_uri> to set one."`)
			return
		}
		homeDir, _ := os.UserHomeDir()
		confPath := filepath.Join(homeDir, ".ssh", "config")
		backupConfPath := filepath.Join(homeDir, ".ssh", "config.bak")

		ok1, _ := utils.PathIsExist(confPath)
		ok2, _ := utils.PathIsExist(backupConfPath)

		if ok1 && ok2 {
			os.RemoveAll(backupConfPath)
		}

		content := []byte{}
		if ok1 {
			content, _ = os.ReadFile(confPath)
		} else if ok2 {
			content, _ = os.ReadFile(backupConfPath)
		}

		if !strings.Contains(string(content), "ProxyCommand") {
			that.SetProxyForGitSSH(pxyURI)
		} else if !ok1 && ok2 {
			that.ToggleProxyForGitSSH()
		}
	}
	// start lazygit
	var (
		commit      string = "5e388e2"
		date        string = "2023-08-07"
		version     string = "0.40.2"
		buildSource string = "gvc"
	)
	ldFlagsBuildInfo := &lazyapp.BuildInfo{
		Commit:      commit,
		Date:        date,
		Version:     version,
		BuildSource: buildSource,
	}
	oldArgs := os.Args
	os.Args = append([]string{"lg"}, args...)
	lazyapp.Start(ldFlagsBuildInfo, nil)
	os.Args = oldArgs

	if enableProxy {
		that.ToggleProxyForGitSSH()
	}
}

/*
Pushes .ssh files to remote repo.
Pulls .ssh files from remote repo.
*/
func (that *GhDownloader) HandleDotSSHFiles(toDownload bool) {
	localSSHDir := filepath.Join(utils.GetHomeDir(), ".ssh")
	remoteFileName := "dotssh.zip"

	repoSyncer := NewSynchronizer()
	if toDownload {
		// download and deploy.
		repoSyncer.DownloadFile(
			localSSHDir,
			remoteFileName,
			EncryptByZip,
		)
	} else {
		// zip and upload.
		repoSyncer.UploadFile(
			localSSHDir,
			remoteFileName,
			EncryptByZip,
		)
	}
}
