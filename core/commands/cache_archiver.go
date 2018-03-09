package commands

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/core/archives"
	"gitlab.com/gitlab-org/gitlab-runner/core/formatter"
	"gitlab.com/gitlab-org/gitlab-runner/core/network"
)

type CacheArchiverCommand struct {
	fileArchiver
	retryHelper
	File    string `long:"file" description:"The path to file"`
	URL     string `long:"url" description:"URL of remote cache resource"`
	Timeout int    `long:"timeout" description:"Overall timeout for cache uploading request (in minutes)"`

	client *network.CacheClient
}

func (c *CacheArchiverCommand) getClient() *network.CacheClient {
	if c.client == nil {
		c.client = network.NewCacheClient(c.Timeout)
	}

	return c.client
}

func (c *CacheArchiverCommand) upload() (bool, error) {
	logrus.Infoln("Uploading", filepath.Base(c.File), "to", formatter.CleanURL(c.URL))

	file, err := os.Open(c.File)
	if err != nil {
		return false, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("PUT", c.URL, file)
	if err != nil {
		return true, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	req.ContentLength = fi.Size()

	resp, err := c.getClient().Do(req)
	if err != nil {
		return true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		// Retry on server errors
		retry := resp.StatusCode/100 == 5
		return retry, fmt.Errorf("Received: %s", resp.Status)
	}

	return false, nil
}

func (c *CacheArchiverCommand) Execute(*cli.Context) {
	if c.File == "" {
		logrus.Fatalln("Missing --file")
	}

	// Enumerate files
	err := c.enumerate()
	if err != nil {
		logrus.Fatalln(err)
	}

	// Check if list of files changed
	if !c.isFileChanged(c.File) {
		logrus.Infoln("Archive is up to date!")
		return
	}

	// Create archive
	err = archives.CreateZipFile(c.File, c.sortedFiles())
	if err != nil {
		logrus.Fatalln(err)
	}

	// Upload archive if needed
	if c.URL != "" {
		err := c.doRetry(c.upload)
		if err != nil {
			logrus.Fatalln(err)
		}
	}
}

func init() {
	RegisterCommand2("cache-archiver", "create and upload cache artifacts (internal)", &CacheArchiverCommand{
		retryHelper: retryHelper{
			Retry:     2,
			RetryTime: time.Second,
		},
	})
}
