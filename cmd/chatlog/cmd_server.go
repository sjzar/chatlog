package chatlog

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/sjzar/chatlog/internal/chatlog"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentPreRun = initLog
	serverCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "debug")
	serverCmd.Flags().StringVarP(&serverAddr, "addr", "a", "", "server address")
	serverCmd.Flags().StringVarP(&serverPlatform, "platform", "p", "", "platform")
	serverCmd.Flags().IntVarP(&serverVer, "version", "v", 0, "version")
	serverCmd.Flags().StringVarP(&serverDataDir, "data-dir", "d", "", "data dir")
	serverCmd.Flags().StringVarP(&serverDataKey, "data-key", "k", "", "data key")
	serverCmd.Flags().StringVarP(&serverImgKey, "img-key", "i", "", "img key")
	serverCmd.Flags().StringVarP(&serverWorkDir, "work-dir", "w", "", "work dir")
	serverCmd.Flags().BoolVarP(&serverAutoDecrypt, "auto-decrypt", "", false, "auto decrypt")
}

var (
	serverAddr        string
	serverDataDir     string
	serverDataKey     string
	serverImgKey      string
	serverWorkDir     string
	serverPlatform    string
	serverVer         int
	serverAutoDecrypt bool
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server",
	Run: func(cmd *cobra.Command, args []string) {

		cmdConf := getServerConfig()
		log.Info().Msgf("server cmd config: %+v", cmdConf)

		m := chatlog.New()
		if err := m.CommandHTTPServer("", cmdConf); err != nil {
			log.Err(err).Msg("failed to start server")
			return
		}
	},
}

func getServerConfig() map[string]any {
	cmdConf := make(map[string]any)
	if len(serverAddr) != 0 {
		cmdConf["http_addr"] = serverAddr
	}
	if len(serverDataDir) != 0 {
		cmdConf["data_dir"] = serverDataDir
	}
	if len(serverDataKey) != 0 {
		cmdConf["data_key"] = serverDataKey
	}
	if len(serverImgKey) != 0 {
		cmdConf["img_key"] = serverImgKey
	}
	if len(serverWorkDir) != 0 {
		cmdConf["work_dir"] = serverWorkDir
	}
	if len(serverPlatform) != 0 {
		cmdConf["platform"] = serverPlatform
	}
	if serverVer != 0 {
		cmdConf["version"] = serverVer
	}
	if serverAutoDecrypt {
		cmdConf["auto_decrypt"] = true
	}
	return cmdConf
}
