package chatlog

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/export"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "export format (json/csv)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path")
	exportCmd.Flags().StringVarP(&exportTimeRange, "time", "t", "", "time range (YYYY-MM-DD~YYYY-MM-DD)")
	exportCmd.Flags().StringVarP(&exportTalker, "talker", "k", "", "chat target (wxid/group id/nickname)")
	exportCmd.Flags().StringVarP(&exportDataDir, "data-dir", "d", "", "data directory")
	exportCmd.Flags().StringVarP(&exportWorkDir, "work-dir", "w", "", "work directory")
	exportCmd.Flags().StringVarP(&exportPlatform, "platform", "p", "", "platform (windows/darwin)")
	exportCmd.Flags().IntVarP(&exportVersion, "version", "v", 0, "version (3/4)")
	exportCmd.Flags().StringVarP(&exportKey, "key", "y", "", "decryption key")
}

var (
	exportFormat    string
	exportOutput    string
	exportTimeRange string
	exportTalker    string
	exportDataDir   string
	exportWorkDir   string
	exportPlatform  string
	exportVersion   int
	exportKey       string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export chat logs",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := chatlog.New("")
		if err != nil {
			log.Err(err).Msg("failed to create chatlog instance")
			return
		}

		// 设置工作目录和数据目录
		if exportDataDir == "" {
			log.Error().Msg("data directory is required")
			return
		}
		if exportWorkDir == "" {
			exportWorkDir = util.DefaultWorkDir(filepath.Base(filepath.Dir(exportDataDir)))
		}
		if exportPlatform == "" {
			log.Error().Msg("platform is required")
			return
		}
		if exportVersion == 0 {
			log.Error().Msg("version is required")
			return
		}
		if exportKey == "" {
			log.Error().Msg("decryption key is required")
			return
		}

		// 设置参数
		if err := m.CommandDecrypt(exportDataDir, exportWorkDir, exportKey, exportPlatform, exportVersion); err != nil {
			log.Err(err).Msg("failed to set parameters")
			return
		}

		// 解析时间范围
		var startTime, endTime time.Time
		if exportTimeRange != "" {
			times := strings.Split(exportTimeRange, "~")
			if len(times) == 2 {
				startTime, _ = time.Parse("2006-01-02", times[0])
				endTime, _ = time.Parse("2006-01-02", times[1])
				endTime = endTime.Add(24 * time.Hour) // 包含结束日期
			}
		}

		// 启动数据库服务
		db := database.NewService(m.Context())
		if err := db.Start(); err != nil {
			log.Err(err).Msg("failed to start database service")
			return
		}
		defer db.Stop()

		// 获取聊天记录
		fmt.Println("正在导出聊天记录")
		fmt.Println("正在获取消息列表...")
		messages, err := export.GetMessagesForExport(db, startTime, endTime, exportTalker, false, func(current, total int) {
			percentage := float64(current) / float64(total) * 100
			width := 30 // 进度条宽度
			completed := int(float64(width) * float64(current) / float64(total))
			remaining := width - completed

			// 构建进度条
			progressBar := fmt.Sprintf("\r获取消息: [%s%s] %.1f%% (%d/%d)",
				strings.Repeat("=", completed),
				strings.Repeat("-", remaining),
				percentage,
				current,
				total)

			fmt.Print(progressBar)
			if current == total {
				fmt.Println() // 完成后换行
			}
		})
		if err != nil {
			log.Err(err).Msg("failed to get messages")
			return
		}

		// 确定输出文件路径
		if exportOutput == "" {
			exportOutput = fmt.Sprintf("chatlog_%s.%s", time.Now().Format("20060102_150405"), exportFormat)
		}

		// 导出消息
		fmt.Println("正在写入文件...")
		if err := export.ExportMessages(messages, exportOutput, exportFormat, func(current, total int) {
			percentage := float64(current) / float64(total) * 100
			width := 30 // 进度条宽度
			completed := int(float64(width) * float64(current) / float64(total))
			remaining := width - completed

			// 构建进度条
			progressBar := fmt.Sprintf("\r写入文件: [%s%s] %.1f%% (%d/%d)",
				strings.Repeat("=", completed),
				strings.Repeat("-", remaining),
				percentage,
				current,
				total)

			fmt.Print(progressBar)
			if current == total {
				fmt.Println() // 完成后换行
			}
		}); err != nil {
			log.Err(err).Msg("failed to export messages")
			return
		}

		fmt.Printf("Successfully exported chat logs to %s\n", exportOutput)
	},
}
