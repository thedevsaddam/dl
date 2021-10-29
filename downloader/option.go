package downloader

import (
	"errors"
	"strings"

	"github.com/thedevsaddam/dl/logger"
	"github.com/thedevsaddam/dl/values"
)

// option describes type for providing configuration options to JSONQ
type option struct {
	concurrency    int
	path           string                   // directory
	subPathMap     values.MapStrSliceString // sub directory
	skipSubPathMap bool
	log            logger.Logger
	verbose        bool
}

// OptionFunc represents a contract for option func, it basically set options to jsonq instance options
type OptionFunc func(*DownloadManager) error

// WithConcurrency set number of concurrent request to be made
func WithConcurrency(c uint) OptionFunc {
	return func(dm *DownloadManager) error {
		if c == 0 {
			return errors.New("dl: concurrency can't be empty")
		}
		dm.option.concurrency = int(c)
		return nil
	}
}

// WithFilePath set the directory to save file
func WithFilePath(path string) OptionFunc {
	return func(dm *DownloadManager) error {
		if path == "" {
			return errors.New("dl: filepath can't be empty")
		}
		dm.option.path = strings.TrimSpace(strings.TrimSuffix(path, "/"))
		return nil
	}
}

// WithSubPathMap set the directory to save file
func WithSubPathMap(m values.MapStrSliceString) OptionFunc {
	return func(dm *DownloadManager) error {
		dm.option.subPathMap = m
		return nil
	}
}

// WithSkipSubPathMap skip subdirectory resolving
func WithSkipSubPathMap() OptionFunc {
	return func(dm *DownloadManager) error {
		dm.option.skipSubPathMap = true
		return nil
	}
}

// WithFilename set the filename
func WithFilename(name string) OptionFunc {
	return func(dm *DownloadManager) error {
		if name == "" {
			return errors.New("dl: filename can't be empty")
		}
		dm.fileName = strings.TrimSpace(name)
		return nil
	}
}

// WithVerbose enable debug output log
func WithVerbose() OptionFunc {
	return func(dm *DownloadManager) error {
		dm.option.verbose = true
		return nil
	}
}

// WithLogger set custom logger
func WithLogger(l logger.Logger) OptionFunc {
	return func(dm *DownloadManager) error {
		dm.option.log = l
		return nil
	}
}
