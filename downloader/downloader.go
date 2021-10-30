package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/thedevsaddam/dl/logger"
	"github.com/thedevsaddam/retry"
)

const (
	defaultConcurrency = 5
)

// DownloadManager ...
type DownloadManager struct {
	option option
	client HTTPClient

	stop      chan os.Signal
	completed chan bool

	fileName            string        // filename with extension
	fileSize            uint64        // file size in bytes
	totalDownloaded     uint64        // total file downloaded in bytes
	totalChunkCompleted int32         // total completed chunks
	totalTimeTaken      time.Duration // total time taken to complete downloading

	errors []error // contains all the errors

	mu *sync.Mutex
	wg *sync.WaitGroup
}

// New return a instance of DownloadManager with options
func New(options ...OptionFunc) *DownloadManager {
	dm := &DownloadManager{
		wg:     &sync.WaitGroup{},
		mu:     &sync.Mutex{},
		client: http.DefaultClient,

		errors: make([]error, 0),

		stop:      make(chan os.Signal, 1),
		completed: make(chan bool, 1),
	}

	// set default options
	dm.option.concurrency = defaultConcurrency
	dm.option.log = logger.New(dm.option.verbose) // enable verbose for applying options

	// apply user provided options
	for _, option := range options {
		if err := option(dm); err != nil {
			fmt.Println("Error:", err)
		}
	}

	return dm
}

// ApplyOption apply single option to the download manager
func (d *DownloadManager) ApplyOption(option OptionFunc) error {
	return option(d)
}

// addError add error to the error bag
func (d *DownloadManager) addError(e error) {
	d.mu.Lock()
	d.errors = append(d.errors, e)
	d.mu.Unlock()
}

// Errors return the error bag
func (d *DownloadManager) Errors() []error {
	return d.errors
}

// renderProgressBar paint & repaint progressbar
func (d *DownloadManager) renderProgressBar(ctx context.Context, maxSize int) {
	pb := progressbar.NewOptions(maxSize,
		progressbar.OptionSetDescription(fmt.Sprintf("[red][0/%d][reset] Downloading:", d.option.concurrency)),
		progressbar.OptionFullWidth(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowCount(),
		progressbar.OptionThrottle(500*time.Millisecond),
		progressbar.OptionOnCompletion(
			func() {
				fmt.Printf("\nFile name: %s\n", d.fileName)
				fmt.Printf("File size: %s\n", humanaReadableBytes(float64(d.fileSize)))
				fmt.Printf("Total time: %s\n", d.totalTimeTaken)
				fmt.Printf("Open: %s/%s\n", d.option.path, d.fileName)
			},
		),
	)

	ticker := time.NewTicker(500 * time.Millisecond)

	go func() {
		for {
			select {
			case <-d.stop:
				d.option.log.Println("Operation cancelled!")
				fmt.Printf("\nOperation cancelled!\n")
				ticker.Stop()
				os.Exit(1)
			case <-d.completed:
				ticker.Stop()
				d.option.log.Println("Download completed!")
				return
			case <-ticker.C:
				pb.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Downloading:", d.totalChunkCompleted, d.option.concurrency))
				pb.Set64(int64(d.totalDownloaded))
			}
		}
	}()
}

// GetFileName return file name; the value will be available once the download start
func (d *DownloadManager) GetFileName() string {
	return d.fileName
}

// GetFileSize return file size in human readable format; the value will be available once the download start
func (d *DownloadManager) GetFileSize() string {
	return humanaReadableBytes(float64(d.fileSize))
}

// populateFileInfo do a HTTP/HEAD request and gather meta information; MUST call before download
func (d *DownloadManager) populateFileInfo(ctx context.Context, url string) error {
	if d.fileName == "" {
		d.fileName = path.Base(url)
	}

	d.option.log.Printf("Info: fetching file information: %s\n", url)

	err := retry.DoFunc(5, 1*time.Second, func() error {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			d.option.log.Printf("Error: failed to create HTTP/HEAD request: %s\n", err.Error())
			return err
		}

		resp, err := d.client.Do(req)
		if err != nil {
			d.option.log.Printf("Error: failed to perform HTTP/HEAD request: %s\n", err.Error())
			return err
		}

		d.fileSize = uint64(resp.ContentLength)

		return resp.Body.Close()
	})

	return err
}

// Download download files based on configurations
func (d *DownloadManager) Download(url string) *DownloadManager {
	signal.Notify(d.stop, syscall.SIGKILL, syscall.SIGINT, syscall.SIGQUIT)
	startedAt := time.Now()
	fmt.Println()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := d.populateFileInfo(ctx, url); err != nil { // populate file information for future use
		d.addError(err)
		cancel()
		return d
	}

	chunkLen := int(d.fileSize) / d.option.concurrency
	rem := int(d.fileSize) % d.option.concurrency

	fileName := d.fileName
	if d.option.path != "" {
		d.option.log.Printf("Info: Root directory: %s\n", d.option.path)

		if !d.option.skipSubPathMap {
			subPath := "other"
			if sp := d.option.subPathMap.Get(filepath.Ext(d.fileName)); sp != "" {
				subPath = sp
			}

			makeSubDir := fmt.Sprintf("%s/%s", d.option.path, subPath)
			if _, err := os.Stat(makeSubDir); os.IsNotExist(err) {
				if err := os.MkdirAll(makeSubDir, os.ModePerm); err != nil {
					d.option.log.Printf("Error: failed to create sub-directory: %s\n", err.Error())
					d.addError(err)
					cancel()
					return d
				}
				d.option.log.Printf("Info: Created sub-directory: %s\n", subPath)
			}

			fileName = fmt.Sprintf("%s/%s/%s", d.option.path, subPath, d.fileName)
		}

		fileName = fmt.Sprintf("%s/%s", d.option.path, d.fileName)
	}

	if _, err := os.Create(fileName); err != nil {
		d.option.log.Printf("Error: failed to create file: %s\n", err.Error())
		d.addError(err)
		cancel()
		return d
	}
	d.option.log.Printf("Info: Created file: %s\n", fileName)

	d.option.log.Printf("Downloading file with concurrency value: %d\n", d.option.concurrency)
	for i := 0; i < d.option.concurrency; i++ {
		d.wg.Add(1)

		min := chunkLen * i
		max := chunkLen * (i + 1)
		if i == d.option.concurrency-1 {
			max += rem
		}

		go func(ctx context.Context, dm *DownloadManager, url string, min, max, i int) {
			defer dm.wg.Done()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				dm.option.log.Printf("Error[%d]: failed to create HTTP/GET request: %s\n", i, err.Error())
				dm.addError(err)
				if err != context.Canceled {
					cancel()
				}
				return
			}

			rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1)
			req.Header.Add("Range", rangeHeader)
			resp, err := dm.client.Do(req)
			if err != nil {
				dm.option.log.Printf("Error[%d]: failed to perform HTTP/GET request: %s\n", i, err.Error())
				dm.addError(err)
				if err != context.Canceled {
					cancel()
				}
				return
			}
			defer resp.Body.Close()

			f, err := os.OpenFile(fileName, os.O_RDWR, 0644)
			if err != nil {
				d.option.log.Printf("Error[%d]: failed to open file: %s\n", i, err.Error())
				dm.addError(err)
				cancel()
				return
			}
			defer f.Close()

			_, err = f.Seek(int64(min), 0)
			if err != nil {
				dm.option.log.Printf("Error[%d]: failed to seek file: %s\n", i, err.Error())
				dm.addError(err)
				cancel()
				return
			}

			_, err = io.Copy(f, Reader{resp.Body, &d.totalDownloaded})
			if err != nil {
				dm.option.log.Printf("Error[i]: failed to copy file content: %s\n", i, err.Error())
				dm.addError(err)
				cancel()
				return
			}

			atomic.AddInt32(&dm.totalChunkCompleted, 1)
		}(ctx, d, url, min, max, i)
	}

	// run async task for refreshing progressbar
	// if file size is unknown then use infinite progress bar
	if d.fileSize > 0 {
		d.renderProgressBar(ctx, int(d.fileSize))
	} else {
		d.renderProgressBar(ctx, -1)
	}
	d.wg.Wait()
	d.totalTimeTaken = time.Since(startedAt)
	time.Sleep(600 * time.Millisecond)
	fmt.Println()
	d.completed <- true

	close(d.completed)
	close(d.stop)

	return d
}