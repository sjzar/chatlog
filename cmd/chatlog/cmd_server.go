package chatlog

import (
	"runtime"

	"github.com/sjzar/chatlog/internal/chatlog"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverAddr, "addr", "a", "127.0.0.1:5030", "server address")
	serverCmd.Flags().StringVarP(&serverDataDir, "data-dir", "d", "", "data dir")
	serverCmd.Flags().StringVarP(&serverWorkDir, "work-dir", "w", "", "work dir")
	serverCmd.Flags().StringVarP(&serverPlatform, "platform", "p", runtime.GOOS, "platform")
	serverCmd.Flags().IntVarP(&serverVer, "version", "v", 3, "version")
}

var (
	serverAddr     string
	serverDataDir  string
	serverWorkDir  string
	serverPlatform string
	serverVer      int
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := chatlog.New("")
		if err != nil {
			log.Err(err).Msg("failed to create chatlog instance")
			return
		}
		if err := m.CommandHTTPServer(serverAddr, serverDataDir, serverWorkDir, serverPlatform, serverVer); err != nil {
			log.Err(err).Msg("failed to start server")
			return
		}
	},
}
