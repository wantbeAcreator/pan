package oss

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type ObjectInfo struct {
	Key          string
	Name         string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

type Client struct {
	bucket *alioss.Bucket
}

func NewClient() (*Client, error) {
	cli, err := alioss.New(Endpoint, AccessKeyID, AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("create oss client: %w", err)
	}
	bucket, err := cli.Bucket(BucketName)
	if err != nil {
		return nil, fmt.Errorf("get bucket %q: %w", BucketName, err)
	}
	return &Client{bucket: bucket}, nil
}

func (c *Client) ListAll(prefix string) ([]string, error) {
	var keys []string
	marker := ""
	for {
		res, err := c.bucket.ListObjects(
			alioss.Prefix(prefix),
			alioss.Marker(marker),
			alioss.MaxKeys(1000),
		)
		if err != nil {
			return nil, fmt.Errorf("list objects with prefix %q: %w", prefix, err)
		}
		for _, obj := range res.Objects {
			if obj.Key[len(obj.Key)-1] == '/' {
				continue
			}
			keys = append(keys, obj.Key)
		}
		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
	}
	return keys, nil
}

func (c *Client) ListDir(prefix string) ([]ObjectInfo, error) {
	var items []ObjectInfo
	marker := ""
	for {
		res, err := c.bucket.ListObjects(
			alioss.Prefix(prefix),
			alioss.Delimiter("/"),
			alioss.Marker(marker),
			alioss.MaxKeys(1000),
		)
		if err != nil {
			return nil, fmt.Errorf("list objects with prefix %q: %w", prefix, err)
		}

		for _, cp := range res.CommonPrefixes {
			name := strings.TrimPrefix(cp, prefix)
			name = strings.TrimSuffix(name, "/")
			items = append(items, ObjectInfo{
				Key:   cp,
				Name:  name,
				IsDir: true,
			})
		}

		for _, obj := range res.Objects {
			if obj.Key == prefix {
				continue
			}
			if obj.Key[len(obj.Key)-1] == '/' {
				continue
			}
			name := strings.TrimPrefix(obj.Key, prefix)
			items = append(items, ObjectInfo{
				Key:          obj.Key,
				Name:         name,
				Size:         obj.Size,
				LastModified: obj.LastModified,
			})
		}

		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
	}
	return items, nil
}

func (c *Client) DownloadAll(prefix, dstDir string) error {
	keys, err := c.ListAll(prefix)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		fmt.Printf("no files found under prefix %q\n", prefix)
		return nil
	}
	fmt.Printf("downloading %d files from oss://%s/%s*\n", len(keys), BucketName, prefix)

	var wg sync.WaitGroup
	errCh := make(chan error, len(keys))

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			localName := k[len(prefix):]
			localPath := filepath.Join(dstDir, localName)
			if dir := filepath.Dir(localPath); dir != "." {
				os.MkdirAll(dir, 0755)
			}
			if err := c.bucket.GetObjectToFile(k, localPath); err != nil {
				errCh <- fmt.Errorf("download %s: %w", k, err)
				return
			}
			fmt.Printf("  ok  %s\n", localName)
		}(key)
	}
	wg.Wait()
	close(errCh)

	var firstErr error
	for e := range errCh {
		if firstErr == nil {
			firstErr = e
		}
		fmt.Fprintln(os.Stderr, e)
	}
	return firstErr
}

func (c *Client) DownloadFile(key, localPath string) error {
	if dir := filepath.Dir(localPath); dir != "." {
		os.MkdirAll(dir, 0755)
	}
	return c.bucket.GetObjectToFile(key, localPath)
}

func (c *Client) DownloadFileWithProgress(key, localPath string, onProgress func(downloaded, total int64)) error {
	if dir := filepath.Dir(localPath); dir != "." {
		os.MkdirAll(dir, 0755)
	}
	body, err := c.bucket.GetObject(key)
	if err != nil {
		return err
	}
	defer body.Close()

	totalSize, _ := c.Size(key)
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			f.Write(buf[:n])
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, totalSize)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func (c *Client) Upload(localPath, remoteKey string) error {
	fmt.Printf("uploading %s -> oss://%s/%s\n", localPath, BucketName, remoteKey)
	err := c.bucket.PutObjectFromFile(remoteKey, localPath)
	if err != nil {
		return fmt.Errorf("upload %s: %w", localPath, err)
	}
	fmt.Println("  ok")
	return nil
}

func (c *Client) UploadWithProgress(localPath, remoteKey string, onProgress func(uploaded, total int64)) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	totalSize := fi.Size()

	pr := &progressReader{
		reader:     f,
		total:      totalSize,
		onProgress: onProgress,
	}
	return c.bucket.PutObject(remoteKey, pr)
}

type progressReader struct {
	reader     io.Reader
	total      int64
	uploaded   int64
	onProgress func(uploaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.uploaded += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.uploaded, pr.total)
	}
	return n, err
}

func (c *Client) Delete(key string) error {
	return c.bucket.DeleteObject(key)
}

func (c *Client) DeleteBatch(keys []string) error {
	_, err := c.bucket.DeleteObjects(keys)
	return err
}

func (c *Client) Copy(srcKey, dstKey string) error {
	_, err := c.bucket.CopyObject(srcKey, dstKey)
	return err
}

func (c *Client) PutDir(prefix string) error {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return c.bucket.PutObject(prefix, bytes.NewReader([]byte{}))
}

func (c *Client) Size(key string) (int64, error) {
	header, err := c.bucket.GetObjectMeta(key)
	if err != nil {
		return 0, err
	}
	var size int64
	fmt.Sscanf(header.Get("Content-Length"), "%d", &size)
	return size, nil
}
