package chatlog

import (
	"fmt"

	"github.com/sjzar/chatlog/internal/chatlog"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.Flags().IntVarP(&keyPID, "pid", "p", 0, "pid")
	keyCmd.Flags().BoolVarP(&keyForce, "force", "f", false, "force")
	keyCmd.Flags().BoolVarP(&keyShowXorKey, "xor-key", "x", false, "show xor key")
}

var (
	keyPID        int
	keyForce      bool
	keyShowXorKey bool
)
var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "key",
	Run: func(cmd *cobra.Command, args []string) {
		m := chatlog.New()
		ret, err := m.CommandKey("", keyPID, keyForce, keyShowXorKey)
		if err != nil {
			log.Err(err).Msg("failed to get key")
			return
		}
		fmt.Println(ret)
	},
}
