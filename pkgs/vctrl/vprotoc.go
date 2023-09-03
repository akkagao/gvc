package vctrl

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	tui "github.com/moqsien/goutils/pkgs/gtui"
	"github.com/moqsien/goutils/pkgs/request"
	config "github.com/moqsien/gvc/pkgs/confs"
	"github.com/moqsien/gvc/pkgs/utils"
)

type VProtoBuffer struct {
	Conf    *config.GVConfig
	fetcher *request.Fetcher
	env     *utils.EnvsHandler
	checker *SumChecker
}

func NewProtobuffer() (p *VProtoBuffer) {
	p = &VProtoBuffer{
		Conf:    config.New(),
		fetcher: request.NewFetcher(),
		env:     utils.NewEnvsHandler(),
	}
	p.checker = NewSumChecker(p.Conf)
	p.env.SetWinWorkDir(config.GVCWorkDir)
	return
}

func (that *VProtoBuffer) Install(force bool) {
	key := runtime.GOOS
	if runtime.GOOS == utils.Linux {
		key = fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	}
	that.fetcher.Url = that.Conf.Protobuf.GitlabUrls[key]
	if that.fetcher.Url != "" {
		that.fetcher.Timeout = 20 * time.Minute
		that.fetcher.SetThreadNum(2)
		fPath := filepath.Join(config.ProtobufDir, "protobuf.zip")
		dstDir := filepath.Join(config.ProtobufDir, "protobuf")
		if err := that.fetcher.DownloadAndDecompress(fPath, dstDir, force); err == nil {
			that.CheckAndInitEnv(dstDir)
			tui.PrintSuccess(fPath)
		} else {
			os.RemoveAll(fPath)
			os.RemoveAll(dstDir)
			tui.PrintError(err)
		}
	}
}

func (that *VProtoBuffer) CheckAndInitEnv(protobufDir string) {
	var binPath string
	if dirList, err := os.ReadDir(protobufDir); err == nil {
		for _, d := range dirList {
			if d.IsDir() && d.Name() == "bin" {
				binPath = filepath.Join(protobufDir, d.Name())
				break
			}
		}
	}
	if binPath == "" {
		return
	}
	if runtime.GOOS != utils.Windows {
		protoEnv := fmt.Sprintf(utils.ProtoEnv, binPath)
		that.env.UpdateSub(utils.SUB_PROTOC, protoEnv)
	} else {
		envList := map[string]string{
			"PATH": binPath,
		}
		that.env.SetEnvForWin(envList)
	}
}

func (that *VProtoBuffer) InstallGoProtobufPlugin() {
	if _, err := utils.ExecuteSysCommand(false, "go", "install", that.Conf.Protobuf.ProtoGenGoUrl); err != nil {
		tui.PrintError(err)
	}
}

func (that *VProtoBuffer) InstallGoProtoGRPCPlugin() {
	if _, err := utils.ExecuteSysCommand(false, "go", "install", that.Conf.Protobuf.ProtoGenGRPCUrl); err != nil {
		tui.PrintError(err)
	}
}
