package oss

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

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

func (c *Client) Upload(localPath, remoteKey string) error {
	fmt.Printf("uploading %s -> oss://%s/%s\n", localPath, BucketName, remoteKey)
	err := c.bucket.PutObjectFromFile(remoteKey, localPath)
	if err != nil {
		return fmt.Errorf("upload %s: %w", localPath, err)
	}
	fmt.Println("  ok")
	return nil
}
