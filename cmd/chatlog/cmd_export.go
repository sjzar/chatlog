package chatlog

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog"
	"github.com/sjzar/chatlog/internal/model"
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
		messages, err := db.GetMessages(startTime, endTime, exportTalker, "", "", 0, 0)
		if err != nil {
			log.Err(err).Msg("failed to get messages")
			return
		}

		// 确定输出文件路径
		if exportOutput == "" {
			exportOutput = fmt.Sprintf("chatlog_%s.%s", time.Now().Format("20060102_150405"), exportFormat)
		}

		// 根据格式导出
		switch exportFormat {
		case "json":
			if err := exportJSON(messages, exportOutput); err != nil {
				log.Err(err).Msg("failed to export JSON")
				return
			}
		case "csv":
			if err := exportCSV(messages, exportOutput); err != nil {
				log.Err(err).Msg("failed to export CSV")
				return
			}
		default:
			log.Error().Msg("unsupported format")
			return
		}

		fmt.Printf("Successfully exported chat logs to %s\n", exportOutput)
	},
}

func exportJSON(messages []*model.Message, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(messages)
}

func exportCSV(messages []*model.Message, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	headers := []string{"Time", "Talker", "TalkerName", "Sender", "SenderName", "IsSelf", "Type", "Content"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// 写入数据
	for _, msg := range messages {
		record := []string{
			msg.Time.Format("2006-01-02 15:04:05"),
			msg.Talker,
			msg.TalkerName,
			msg.Sender,
			msg.SenderName,
			fmt.Sprintf("%v", msg.IsSelf),
			fmt.Sprintf("%d", msg.Type),
			msg.Content,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
