package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	progressbar "github.com/schollz/progressbar/v3"

	"github.com/PWZER/dssh/config"
)

func (c *Client) ConnectSftp(host *config.Host) (err error) {
	c.sftpClient, err = sftp.NewClient(c.sshClient)
	return err
}

func (c *Client) doUploadFile(localFile, remoteFile string) (err error) {
	info, err := os.Stat(localFile)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	parent := filepath.Dir(remoteFile)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		c.sftpClient.Mkdir(path)
	}

	dstFile, err := c.sftpClient.Create(remoteFile)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	fmt.Println(localFile, "=>", remoteFile)

	progress := progressbar.DefaultBytes(info.Size(), info.Name())
	if _, err = io.Copy(dstFile, io.TeeReader(srcFile, progress)); err == nil {
		if err = c.sftpClient.Chmod(remoteFile, info.Mode()); err != nil {
			return err
		}
		if err = c.sftpClient.Chtimes(remoteFile, time.Now(), info.ModTime()); err != nil {
			return err
		}
	}
	return err
}

func (c *Client) doUploadDir(localDir, remoteDir string) (err error) {
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		remotePath := strings.ReplaceAll(localPath, localDir, remoteDir)
		if info.IsDir() {
			if err = c.sftpClient.MkdirAll(remotePath); err != nil {
				return err
			}
			if err = c.sftpClient.Chmod(remotePath, info.Mode()); err != nil {
				return err
			}
		} else {
			if err = c.doUploadFile(localPath, remotePath); err != nil {
				return err
			}
		}
		return err
	})
}

func (c *Client) Upload(localPath, remotePath string) (err error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if c.sftpClient, err = sftp.NewClient(c.sshClient); err != nil {
		return err
	}
	defer c.sftpClient.Close()

	if stat.IsDir() {
		return c.doUploadDir(localPath, remotePath)
	}
	return c.doUploadFile(localPath, remotePath)
}

func (c *Client) doDownloadFile(remoteFile, localFile string) (err error) {
	info, err := c.sftpClient.Stat(remoteFile)
	if err != nil {
		return err
	}

	srcFile, err := c.sftpClient.Open(remoteFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	fmt.Println(remoteFile, "=>", localFile)

	progress := progressbar.DefaultBytes(info.Size(), info.Name())
	if _, err = io.Copy(dstFile, io.TeeReader(srcFile, progress)); err == nil {
		if err = os.Chmod(localFile, info.Mode()); err != nil {
			return err
		}
		if err = os.Chtimes(localFile, time.Now(), info.ModTime()); err != nil {
			return err
		}
	}
	return err
}

func (c *Client) doDownloadDir(remoteDir, localDir string) (err error) {
	walker := c.sftpClient.Walk(remoteDir)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		remotePath := walker.Path()
		localPath := strings.ReplaceAll(remotePath, remoteDir, localDir)

		info := walker.Stat()
		if info.IsDir() {
			if err = os.MkdirAll(localPath, info.Mode()); err != nil {
				return err
			}
		} else {
			if err = c.doDownloadFile(remotePath, localPath); err != nil {
				return err
			}
		}
	}
	return err
}

func (c *Client) Download(remotePath, localPath string) (err error) {
	if c.sftpClient, err = sftp.NewClient(c.sshClient); err != nil {
		return err
	}
	defer c.sftpClient.Close()

	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return c.doDownloadDir(remotePath, localPath)
	}
	return c.doDownloadFile(remotePath, localPath)
}

func (c *Client) ListFilesAndDirs(remotePath string) (items []string, err error) {
	//if c.sftpClient, err = sftp.NewClient(c.sshClient); err != nil {
	//    return items, err
	//}
	//defer c.sftpClient.Close()
	if c.sftpClient == nil {
		return items, fmt.Errorf("sftpClient is nil")
	}

	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return items, err
	}

	if stat.IsDir() {
		infos, err := c.sftpClient.ReadDir(remotePath)
		if err != nil {
			return items, err
		}
		for _, info := range infos {
			items = append(items, info.Name())
		}
	}
	return items, nil
}
