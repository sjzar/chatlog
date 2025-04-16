package windows

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/sjzar/chatlog/internal/wechat/model"
	"github.com/sjzar/chatlog/pkg/appver"
)

const (
	V3ProcessName = "WeChat"
	V4ProcessName = "Weixin"
	V3DBFile      = `Msg\Misc.db`
	V4DBFile      = `db_storage\session\session.db`
)

// Detector 实现 Windows 平台的进程检测器
type Detector struct{}

// NewDetector 创建一个新的 Windows 检测器
func NewDetector() *Detector {
	return &Detector{}
}

// FindProcesses 查找所有微信进程并返回它们的信息
func (d *Detector) FindProcesses() ([]*model.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		log.Err(err).Msg("获取进程列表失败")
		return nil, err
	}

	var result []*model.Process
	for _, p := range processes {
		name, err := p.Name()
		name = strings.TrimSuffix(name, ".exe")
		if err != nil || (name != V3ProcessName && name != V4ProcessName) {
			continue
		}

		// v4 存在同名进程，需要继续判断 cmdline
		if name == V4ProcessName {
			cmdline, err := p.Cmdline()
			if err != nil {
				log.Err(err).Msg("获取进程命令行失败")
				continue
			}
			if strings.Contains(cmdline, "--") {
				continue
			}
		}

		// 获取进程信息
		procInfo, err := d.getProcessInfo(p)
		if err != nil {
			log.Err(err).Msgf("获取进程 %d 的信息失败", p.Pid)
			continue
		}

		result = append(result, procInfo)
	}

	return result, nil
}

// getProcessInfo 获取微信进程的详细信息
func (d *Detector) getProcessInfo(p *process.Process) (*model.Process, error) {
	procInfo := &model.Process{
		PID:      uint32(p.Pid),
		Status:   model.StatusOffline,
		Platform: model.PlatformWindows,
	}

	// 获取可执行文件路径
	exePath, err := p.Exe()
	if err != nil {
		log.Err(err).Msg("获取可执行文件路径失败")
		return nil, err
	}
	procInfo.ExePath = exePath

	// 获取版本信息
	versionInfo, err := appver.New(exePath)
	if err != nil {
		log.Err(err).Msg("获取版本信息失败")
		return nil, err
	}
	procInfo.Version = versionInfo.Version
	procInfo.FullVersion = versionInfo.FullVersion

	// 初始化附加信息（数据目录、账户名）
	if err := initializeProcessInfo(p, procInfo); err != nil {
		log.Err(err).Msg("初始化进程信息失败")
		// 即使初始化失败也返回部分信息
	}

	return procInfo, nil
}
