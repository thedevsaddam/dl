package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thedevsaddam/dl/values"
)

const (
	configDirectory = ".dl"
	configFileName  = configDirectory + "/config.json"
)

var (
	defaultConfig Config
)

// Config represent configurations for the download manager
type Config struct {
	Directory   string                   `json:"directory"`
	Concurrency uint                     `json:"concurrency"`
	SubDirMap   values.MapStrSliceString `json:"sub_dir_map"`
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDirectory), nil
}

func getConfigFileName() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configFileName), nil
}

// LoadDefaultConfig load configuration if exist
func LoadDefaultConfig() error {
	fn, err := getConfigFileName()
	if err != nil {
		return err
	}
	if _, err := os.Stat(fn); errors.Is(err, os.ErrNotExist) {
		subDir := make(values.MapStrSliceString)
		subDir["audio"] = []string{".aif", ".cda", ".mid", ".midi", ".mp3",
			".mpa", ".ogg", ".wav", ".wma", ".wpl"}

		subDir["video"] = []string{".3g2", ".3gp", ".avi", ".flv", ".h264",
			".m4v", ".mkv", ".mov", ".mp4", ".mpg", ".mpeg", ".rm", ".swf",
			".vob", ".wmv"}

		subDir["image"] = []string{".ai", ".bmp", ".ico", ".jpeg", ".jpg",
			".png", ".ps", ".psd", ".svg", ".tif", ".tiff"}

		subDir["document"] = []string{".xls", ".xlsm", ".xlsx", ".ods", ".doc",
			".odt", ".pdf", ".rtf", ".tex", ".txt", ".wpd", ".md"}

		CreateConfig(Config{
			Concurrency: 5,
			Directory:   "",
			SubDirMap:   subDir,
		})
	}
	contents, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	return json.Unmarshal(contents, &defaultConfig)
}

// CreateConfig create a config file in user home directory
func CreateConfig(c Config) error {
	bb, err := json.Marshal(c)
	if err != nil {
		return err
	}

	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0777); err != nil {
		return err
	}

	fn, err := getConfigFileName()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fn, bb, 0644)
}

// SetConfig set specific config file in user home directory
func SetConfig(c Config) error {
	oldCfg := defaultConfig
	if c.Concurrency != 0 {
		oldCfg.Concurrency = c.Concurrency
	}

	if c.Directory != "" {
		oldCfg.Directory = c.Directory
	}

	for k, extensions := range c.SubDirMap {
		for _, e := range extensions {
			oldCfg.SubDirMap.Add(k, e)
		}
	}

	return CreateConfig(oldCfg)
}

// DefaultConfig return the default config
func DefaultConfig() Config {
	return defaultConfig
}
