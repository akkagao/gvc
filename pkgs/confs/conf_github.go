package confs

import (
	"fmt"
	"net/http"
	"time"
)

type GithubConf struct {
	DownProxy  string            `koanf:"down_proxy"`
	AccelUrls  []string          `koanf:"acceleration_urls"`
	WinGitUrls map[string]string `koanf:"win_git_urls"`
}

func NewGithubConf() (ghc *GithubConf) {
	ghc = &GithubConf{
		DownProxy: "https://ghproxy.com/",
	}
	return
}

func (that *GithubConf) Reset() {
	that.DownProxy = "https://ghproxy.com/"
	that.AccelUrls = []string{
		"https://ghproxy.com/",
		"https://d.serctl.com/?dl_start",
	}
	that.WinGitUrls = map[string]string{
		"amd64": "https://gitlab.com/moqsien/gvc_resources/-/raw/main/PortableGit-2.42.0.2-64-bit.7z.exe",
		"386":   "https://gitlab.com/moqsien/gvc_resources/-/raw/main/PortableGit-2.42.0.2-32-bit.7z.exe",
	}
}

func (that *GithubConf) testDownProxy() (r bool) {
	if _, err := (&http.Client{Timeout: 20 * time.Second}).Get(that.DownProxy); err == nil {
		r = true
	}
	return
}

func (that *GithubConf) GetDownUrl(oUrl string) (nUrl string) {
	nUrl = oUrl
	if that.testDownProxy() {
		nUrl = fmt.Sprintf("%s%s", that.DownProxy, oUrl)
	}
	return
}
