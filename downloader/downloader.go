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

	"github.com/briandowns/spinner"
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

	fileName            string // filename with extension
	fileSize            uint64 // file size in bytes
	totalDownloaded     uint64 // total file downloaded in bytes
	totalChunkCompleted int32  // total completed chunks
	location            string // where the file stored

	totalTimeTaken time.Duration // total time taken to complete downloading

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
				fmt.Printf("Time elapsed: %s\n", d.totalTimeTaken)
				fmt.Printf("Location: %s\n", d.location)
			},
		),
	)

	ticker := time.NewTicker(500 * time.Millisecond)

	go func() {
		for {
			select {
			case <-d.stop:
				ticker.Stop()
				return
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

	d.option.log.Printf("Info: fetching file's meta information: %s\n", url)
	retryCount := 0

	err := retry.DoFunc(20, 200*time.Millisecond, func() error {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		if retryCount > 0 {
			d.option.log.Printf("Info: retrying to fetch file's meta information [%d]", retryCount)
		}
		retryCount++

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

// downloadChunk download single chunk from the range
func (d *DownloadManager) downloadChunk(ctx context.Context, url string, min, max, chunkNo int, errCh chan error) {
	defer d.wg.Done()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		d.option.log.Printf("Error[%d]: failed to create HTTP/GET request: %s\n", chunkNo, err.Error())
		errCh <- err
		return
	}

	rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1)
	req.Header.Add("Range", rangeHeader)
	resp, err := d.client.Do(req)
	if err != nil {
		d.option.log.Printf("Error[%d]: failed to perform HTTP/GET request: %s\n", chunkNo, err.Error())
		errCh <- err
		return
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(d.location, os.O_RDWR, 0644)
	if err != nil {
		d.option.log.Printf("Error[%d]: failed to open file: %s\n", chunkNo, err.Error())
		errCh <- err
		return
	}
	defer f.Close()

	_, err = f.Seek(int64(min), 0)
	if err != nil {
		d.option.log.Printf("Error[%d]: failed to seek file: %s\n", chunkNo, err.Error())
		errCh <- err
		return
	}

	_, err = io.Copy(f, Reader{resp.Body, &d.totalDownloaded})
	if err != nil {
		d.option.log.Printf("Error[%d]: failed to copy file content: %s\n", chunkNo, err.Error())
		errCh <- err
		return
	}

	atomic.AddInt32(&d.totalChunkCompleted, 1)
}

// Download download files based on configurations
func (d *DownloadManager) Download(url string) *DownloadManager {
	signal.Notify(d.stop, syscall.SIGKILL, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		for range d.stop {
			d.option.log.Println("Operation cancelled!")
			fmt.Printf("\nOperation cancelled!\n")
			// make cursor visible if interruption happened while fetching meta
			io.WriteString(os.Stdout, "\033[?25h")
			os.Exit(1)
		}
	}()

	startedAt := time.Now()
	fmt.Println()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := spinner.New(spinner.CharSets[70], 100*time.Millisecond, spinner.WithHiddenCursor(true)) // code:39 is earth for the lib
	s.Prefix = "Fetching file's meta information ( "
	s.Suffix = ")"
	s.Start()
	if err := d.populateFileInfo(ctx, url); err != nil {
		d.addError(err)
		cancel()
		return d
	}
	s.Stop()
	fmt.Println()

	chunkLen := int(d.fileSize) / d.option.concurrency
	rem := int(d.fileSize) % d.option.concurrency

	fileName := d.fileName
	if d.option.path != "" {
		d.option.log.Printf("Info: Root directory: %s\n", d.option.path)

		fileName = fmt.Sprintf("%s/%s", d.option.path, d.fileName)

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
	}
	d.location = fileName // set location value

	if _, err := os.Create(fileName); err != nil {
		d.option.log.Printf("Error: failed to create file: %s\n", err.Error())
		d.addError(err)
		cancel()
		return d
	}
	d.option.log.Printf("Info: Created file: %s\n", fileName)

	d.option.log.Printf("Downloading file with concurrency value: %d\n", d.option.concurrency)

	errsCh := make(chan error, d.option.concurrency)
	defer close(errsCh)

	// read errors
	go func() {
		for {
			if err := <-errsCh; err != nil {
				d.addError(err)
				if err != context.Canceled {
					cancel()
				}
			}
		}
	}()

	for i := 0; i < d.option.concurrency; i++ {
		d.wg.Add(1)
		min := chunkLen * i
		max := chunkLen * (i + 1)
		if i == d.option.concurrency-1 {
			max += rem
		}
		go d.downloadChunk(ctx, url, min, max, i, errsCh)
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
