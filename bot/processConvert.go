package bot

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func convertAudioToMP3(ctx context.Context, srcPath, dstPath string) error {
	tmpPath := dstPath + ".tmp.mp3"
	_ = os.Remove(tmpPath)
	cmdCtx, cancel := commandContext(ctx)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "ffmpeg", "-y", "-threads", "0", "-i", srcPath, "-vn", "-codec:a", "libmp3lame", "-q:a", "2", tmpPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		if cmdCtx.Err() != nil {
			return fmt.Errorf("转换超时或任务已停止: %w", cmdCtx.Err())
		}
		return fmt.Errorf("转换失败 %s: %s", filepath.Base(srcPath), strings.TrimSpace(string(output)))
	}
	return os.Rename(tmpPath, dstPath)
}

func commandContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	timeout := downloadStallTimeout()
	if timeout < 5*time.Minute {
		timeout = 5 * time.Minute
	}
	return context.WithTimeout(parent, timeout)
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func createZipArchive(archivePath string, files []string) error {
	archive, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	for _, filePath := range files {
		if err := addFileToZip(zipWriter, filePath); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(filePath)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}
