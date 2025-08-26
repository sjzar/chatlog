package chatlog

import (
	"fmt"

	"github.com/sjzar/chatlog/internal/chatlog"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(decryptCmd)
	decryptCmd.Flags().StringVarP(&decryptPlatform, "platform", "p", "", "platform")
	decryptCmd.Flags().IntVarP(&decryptVer, "version", "v", 0, "version")
	decryptCmd.Flags().StringVarP(&decryptDataDir, "data-dir", "d", "", "data dir")
	decryptCmd.Flags().StringVarP(&decryptDatakey, "data-key", "k", "", "data key")
	decryptCmd.Flags().StringVarP(&decryptWorkDir, "work-dir", "w", "", "work dir")
}

var (
	decryptPlatform string
	decryptVer      int
	decryptDataDir  string
	decryptDatakey  string
	decryptWorkDir  string
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "decrypt",
	Run: func(cmd *cobra.Command, args []string) {

		cmdConf := getDecryptConfig()

		m := chatlog.New()
		if err := m.CommandDecrypt("", cmdConf); err != nil {
			log.Err(err).Msg("failed to decrypt")
			return
		}
		fmt.Println("decrypt success")
	},
}

func getDecryptConfig() map[string]any {
	cmdConf := make(map[string]any)
	if len(decryptDataDir) != 0 {
		cmdConf["data_dir"] = decryptDataDir
	}
	if len(decryptDatakey) != 0 {
		cmdConf["data_key"] = decryptDatakey
	}
	if len(decryptWorkDir) != 0 {
		cmdConf["work_dir"] = decryptWorkDir
	}
	if len(decryptPlatform) != 0 {
		cmdConf["platform"] = decryptPlatform
	}
	if decryptVer != 0 {
		cmdConf["version"] = decryptVer
	}
	return cmdConf
}
