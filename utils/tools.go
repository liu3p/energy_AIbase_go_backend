package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// UTF8ToGBK 将UTF-8编码的字节切片转换为GBK编码
func UTF8ToGBK(utf8Data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(utf8Data), simplifiedchinese.GBK.NewEncoder())
	return io.ReadAll(reader)
}

// GBKToUTF8 将GBK编码的字节切片转换为UTF-8编码
func GBKToUTF8(gbkData []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(gbkData), simplifiedchinese.GBK.NewDecoder())
	return io.ReadAll(reader)
}

func UTF8ToGBKStr(utf8Str string) string {
	out, err := UTF8ToGBK([]byte(utf8Str))
	if err != nil {
		return utf8Str
	}
	return string(out)
}

// GBKToUTF8 将GBK编码的字节切片转换为UTF-8编码
func GBKToUTF8Str(gbkStr string) string {
	out, err := GBKToUTF8([]byte(gbkStr))
	if err != nil {
		return gbkStr
	}
	return string(out)
}

func ConvertToUTF8(str string, decoder *encoding.Decoder) (string, error) {
	utf8Bytes, err := decoder.Bytes([]byte(str))
	if err != nil {
		return "", err
	}
	return string(utf8Bytes), nil
}

type MatchFunc func(filePath string, fileInfo os.FileInfo) bool

// 打包文件
func TarGzDirectory(sourceDir, destinationPath string, matchFunc MatchFunc) error {
	// 创建目标文件
	destFile, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// 创建 Gzip 编写器
	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	// 创建 Tar 编写器
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// 遍历目录并将文件写入 Tar 编写器
	err = filepath.Walk(sourceDir, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filePath == sourceDir {
			// skip sourceDir itself
			return nil
		}

		if !matchFunc(filePath, fileInfo) {
			return nil
		}

		// 构建文件头
		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return err
		}

		// 设置文件在 tar 中的相对路径
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}
		utf8RelPath, err := ConvertToUTF8(relPath, simplifiedchinese.GBK.NewDecoder())
		if err != nil {
			return err
		}
		header.Name = utf8RelPath // relPath

		// 写入文件头到 tar 编写器
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 如果是文件，将文件内容写入 tar 编写器
		if !fileInfo.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory and create tar.gz: %w", err)
	}

	return nil
}
