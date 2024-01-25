package confs

import (
	"os"

	"github.com/moqsien/goutils/pkgs/gtea/gprint"
	"github.com/moqsien/gvc/pkgs/utils"
)

type NUrl struct {
	Url  string `koanf:"url"`
	Name string `koanf:"name"`
	Ext  string `koanf:"ext"`
}

type NVimConf struct {
	NvimUrl     string `koanf,json:"nvim_url"`
	PluginsUrl  string `koanf:"plugins_url"`
	GithubProxy string `koanf:"github_proxy"`
	path        string
}

func NewNVimConf() (r *NVimConf) {
	r = &NVimConf{
		path: NVimFileDir,
	}
	r.setup()
	return
}

func (that *NVimConf) setup() {
	if ok, _ := utils.PathIsExist(that.path); !ok {
		if err := os.MkdirAll(that.path, os.ModePerm); err != nil {
			gprint.PrintError("%+v", err)
		}
	}
}

func (that *NVimConf) Reset() {
	that.NvimUrl = "https://github.com/neovim/neovim/releases/latest/"
	that.PluginsUrl = "https://gitlab.com/moqsien/gvc_resources/uploads/753afef9d38f8f6224d221770d25c9a3/nvim-plugins.zip"
	that.GithubProxy = "https://ghproxy.com/"
}
