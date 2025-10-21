package minio

import (
    "context"
    "io"
    "net/url"
    "os"
    "path"
    "strings"

    minio "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
    cli       *minio.Client
    bucket    string
    publicURL string
}

func New() (*Client, error) {
    endpoint := getenv("MINIO_ENDPOINT", "localhost:9000")
    access := getenv("MINIO_ACCESS_KEY", "minio")
    secret := getenv("MINIO_SECRET_KEY", "minio124")
    bucket := getenv("MINIO_BUCKET", "pictures")
    public := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")

    c, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(access, secret, ""),
        Secure: strings.HasPrefix(endpoint, "https"),
    })
    if err != nil {
        return nil, err
    }
    // ensure bucket exists
    ctx := context.Background()
    exists, err := c.BucketExists(ctx, bucket)
    if err != nil {
        return nil, err
    }
    if !exists {
        if err := c.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
            return nil, err
        }
    }
    return &Client{cli: c, bucket: bucket, publicURL: public}, nil
}

func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
    _, err := c.cli.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{ContentType: contentType})
    if err != nil {
        return "", err
    }
    u := strings.TrimRight(c.publicURL, "/") + "/" + path.Join(c.bucket, objectName)
    return u, nil
}

func (c *Client) DeleteByURL(ctx context.Context, fileURL string) error {
    if fileURL == "" {
        return nil
    }
    // parse last two segments as bucket/object if possible
    u, err := url.Parse(fileURL)
    if err != nil {
        return nil
    }
    parts := strings.Split(strings.Trim(u.Path, "/"), "/")
    if len(parts) < 2 {
        return nil
    }
    bucket := parts[0]
    object := strings.Join(parts[1:], "/")
    return c.cli.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
}

func getenv(k, d string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return d
}


