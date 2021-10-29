package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedevsaddam/dl/config"
)

var (
	path    string
	subPath string

	cmdConfig = &cobra.Command{
		Use:   "config",
		Short: "Config set configuration values",
		Long:  `Config set configuration values`,
		Run:   setConfig,
	}
)

func init() {
	cmdConfig.Flags().StringVarP(&path, "path", "p", "", "destination directory where the file will be downloaded")
	cmdConfig.Flags().StringVarP(&subPath, "subpath", "s", "", "sub directory map in this format subdirectoryName:.ext1,.ext2. e.g: video:.mp4,.mkv")
	cmdConfig.Flags().IntVarP(&concurrent, "concurrent", "c", 0, "number of concurrent process will be running, default: 5")
	cmdConfig.Flags().BoolVarP(&debug, "debug", "d", false, "display configuration")
	cmdDL.AddCommand(cmdConfig)
}

func setConfig(cmd *cobra.Command, args []string) {
	// if user provide "." then set current directory as root directory
	if path == "." {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
		path = dir
	}

	newCfg := config.Config{Directory: path, Concurrency: uint(concurrent)}

	if subPath != "" {
		newCfg.SubDirMap = config.DefaultConfig().SubDirMap // assign old config

		pp := strings.Split(subPath, ":")
		if len(pp) > 1 {
			subDir := pp[0]
			extensions := strings.Split(pp[1], ",")
			for _, e := range extensions {
				newCfg.SubDirMap.Add(subDir, e)
			}
		}
	}

	if err := config.SetConfig(newCfg); err != nil {
		log.Fatalln(err)
	}

	if debug {
		if err := config.LoadDefaultConfig(); err != nil {
			log.Fatalln(err)
		}
		bb, err := json.MarshalIndent(config.DefaultConfig(), "", "  ")
		if err == nil {
			fmt.Println(string(bb))
		}
	}
}
