package chatlog

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Debug bool

func initLog(cmd *cobra.Command, args []string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func initTuiLog(cmd *cobra.Command, args []string) {
	logOutput := io.Discard

	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		logpath := util.DefaultWorkDir("")
		util.PrepareDir(logpath)
		logFD, err := os.OpenFile(filepath.Join(logpath, "chatlog.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			panic(err)
		}
		logOutput = logFD
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logOutput, NoColor: true, TimeFormat: time.RFC3339})
	logrus.SetOutput(logOutput)
}
