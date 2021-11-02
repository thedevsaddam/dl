package cmd

import (
	"context"
	"fmt"
	"log"
	netUrl "net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedevsaddam/dl/config"
	"github.com/thedevsaddam/dl/downloader"
	"github.com/thedevsaddam/dl/logger"
	"github.com/thedevsaddam/dl/notifier"
	"github.com/thedevsaddam/dl/update"
)

var logo = `
  _____  _      
 |  __ \| |     
 | |  | | |     
 | |  | | |     
 | |__| | |____ 
 |_____/|______|
                
Command-line file downloader tool
For more info visit: https://github.com/thedevsaddam/dl      
`

const (
	unknown = "unknown"
)

var (
	url        string
	name       string
	concurrent int
	debug      bool

	GitCommit = unknown
	Version   = "v1.0.2"
	BuildDate = "2021-10-15"

	// cmdDL is the root command of DL application
	cmdDL = &cobra.Command{
		Use:   "dl",
		Short: "Command-line file downloader tool",
		Long:  logo,
		// Args:  cobra.MinimumNArgs(1),
		Run: startDownload,
	}
)

// Execute executes the root command
func Execute() {
	if err := cmdDL.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cmdDL.Flags().StringVarP(&url, "url", "u", "", "url should be the address where the file will be downloaded. e.g: https://example.com/foo.jpg")
	cmdDL.Flags().StringVarP(&name, "name", "n", "", "destination name with extension. e.g: foo.jpg")
	cmdDL.Flags().StringVarP(&path, "path", "p", "", "destination directory where the file will be downloaded")
	cmdDL.Flags().IntVarP(&concurrent, "concurrent", "c", 0, "number of concurrent process will be running, default: 5")
	cmdDL.Flags().BoolVarP(&debug, "debug", "d", false, "debug print the essential logs")
}

func initConfig() {
	err := config.LoadDefaultConfig()
	if err != nil {
		log.Fatalln(err)
	}
}

// startDownload fire the whole download process and orchestrate other dependent processes
func startDownload(cmd *cobra.Command, args []string) {
	cfg := config.DefaultConfig()

	if cfg.AutoUpdate {
		err := update.SelfUpdate(context.Background(), BuildDate, Version)
		if err != nil {
			fmt.Println("Error: failed to update dl:", err) //this error can be skipped
		}
	}

	url = strings.TrimSpace(url)
	if len(url) == 0 {
		return
	}

	if _, err := netUrl.ParseRequestURI(url); err != nil {
		fmt.Println("Error: invalid URL:", err)
		return
	}

	dm := downloader.New()

	if debug {
		dm.ApplyOption(downloader.WithVerbose())
		dm.ApplyOption(downloader.WithLogger(logger.New(true)))
	}

	con := cfg.Concurrency
	if concurrent != 0 { // assuming default is 5
		con = uint(concurrent)
	}
	dm.ApplyOption(downloader.WithConcurrency(uint(con)))

	dm.ApplyOption(downloader.WithSubPathMap(cfg.SubDirMap))

	if cfg.Directory != "" {
		dm.ApplyOption(downloader.WithFilePath(cfg.Directory))
	}

	// if provided in flag then override the pverious path config
	if path != "" {
		// if user provide "." then set current directory as root directory
		if path == "." {
			dir, err := os.Getwd()
			if err != nil {
				log.Fatalln(err)
			}
			path = dir
		}
		dm.ApplyOption(downloader.WithFilePath(path))
		dm.ApplyOption(downloader.WithSkipSubPathMap())
	}

	if name != "" {
		dm.ApplyOption(downloader.WithFilename(name))
	}

	errs := dm.Download(url).Errors()
	if len(errs) > 0 {
		fmt.Printf("\nDownload failed\n")
		if debug {
			for _, e := range errs {
				log.Println("Error:", e)
			}
		}
		os.Exit(1)
		return
	}

	n := notifier.New("DL [Terminal Downloader]")
	n.Notify("Download complete!", fmt.Sprintf("File: %s (%s)", dm.GetFileName(), dm.GetFileSize()))
}
