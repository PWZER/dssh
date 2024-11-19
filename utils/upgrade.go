package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/schollz/progressbar/v3"
)

const (
	latestVersionUrl string = "https://api.github.com/repos/PWZER/dssh/releases/latest"
)

type versionAsset struct {
	Url                string `json:"url"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

type latestVersion struct {
	TagName string         `json:"tag_name"`
	Assets  []versionAsset `json:"assets"`
}

func downloadFileFromURL(saveFile, srcURL string) (err error) {
	fd, err := os.Create(saveFile)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			fd.Close()
			os.Remove(saveFile)
		}
	}()
	c := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  true,
			Proxy:              http.ProxyFromEnvironment,
		},
	}
	res, err := c.Get(srcURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}

	bar := progressbar.DefaultBytes(
		res.ContentLength,
		"Downloading",
	)
	n, err := io.Copy(io.MultiWriter(fd, bar), res.Body)
	if err != nil {
		return err
	}
	if res.ContentLength != -1 && res.ContentLength != n {
		return fmt.Errorf("downloaded size mismatch, expect %d but got %d", res.ContentLength, n)
	}
	return fd.Close()
}

func getLatestVersion() (*latestVersion, error) {
	req, err := http.NewRequest(http.MethodGet, latestVersionUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request, err: %s", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version, err: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get latest version, status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read latest version, err: %s", err)
	}

	var latestVersion latestVersion
	if err := json.Unmarshal(body, &latestVersion); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latest version, err: %s", err)
	}

	return &latestVersion, nil
}

func Upgrade(dummy bool, currentVersion string) (err error) {
	latestVersion, err := getLatestVersion()
	if err != nil {
		return err
	}

	if latestVersion.TagName == "" {
		return fmt.Errorf("got the latest version is empty")
	}

	if latestVersion.TagName <= currentVersion {
		fmt.Printf("already the latest version, or current version: %s is newer than latest version: %s\n",
			currentVersion, latestVersion.TagName)
		return nil
	}

	if dummy {
		fmt.Printf("current version: %s, latest version: %s\n", currentVersion, latestVersion.TagName)
		return nil
	}

	var useAsset *versionAsset = nil
	expectedAssetName := fmt.Sprintf("dssh-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range latestVersion.Assets {
		if asset.Name == expectedAssetName {
			useAsset = &asset
			break
		}
	}

	if useAsset == nil {
		return fmt.Errorf("latest version %s not found asset for %s", latestVersion.TagName, expectedAssetName)
	}
	fmt.Printf("Upgrade version %s => %s, downloading from %s\n",
		currentVersion, latestVersion.TagName, useAsset.BrowserDownloadUrl)

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get binary path, err: %s", err)
	}

	tmpPath := filepath.Join(filepath.Dir(binPath), fmt.Sprintf(".%s.tmp", filepath.Base(binPath)))
	if err := downloadFileFromURL(tmpPath, useAsset.BrowserDownloadUrl); err != nil {
		return fmt.Errorf("failed to download file, err: %s", err)
	}

	// check file size
	stat, err := os.Stat(tmpPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("downloaded file not found: %s", tmpPath)
	} else if stat.Size() != int64(useAsset.Size) {
		return fmt.Errorf("downloaded file size mismatch, expect %d but got %d", useAsset.Size, stat.Size())
	}

	// make it executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to make file executable, err: %s", err)
	}

	// replace the binary
	if err := os.Rename(tmpPath, binPath); err != nil {
		return fmt.Errorf("failed to replace binary, err: %s", err)
	}
	fmt.Printf("upgrade to version %s successfully\n", latestVersion.TagName)
	return nil
}
