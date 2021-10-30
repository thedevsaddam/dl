package notifier

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const (
	configDirectory = ".dl"
	icon            = ".dl" + "/icon.png"
)

func getIconPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, icon), nil
}

func writeIcon() {
	iconPath, _ := getIconPath()
	if _, err := os.Stat(iconPath); err == nil {
		return
	}

	fp, err := AssetFS.Open("/icon.png")
	if err != nil {
		log.Fatal(err)
	}
	bs, err := ioutil.ReadAll(fp)
	if err != nil {
		log.Fatal(err)
	}
	if err == nil {
		err := ioutil.WriteFile(iconPath, bs, 0644)
		if err != nil {
			log.Println(err)
		}
	}
}
