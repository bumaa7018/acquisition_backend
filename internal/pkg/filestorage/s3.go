package filestorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage — MinIO/S3-д тулгуурласан файлын систем.
//
// API нь файлыг ӨӨР ДЭЭРЭЭ ХАДГАЛАХГҮЙ: байршуулсан файл шууд объектын
// хадгалалт руу явж, DB-д зөвхөн object-ийн зам (URL) бичигдэнэ.
//
// URL нь `BaseURL/<key>` хэлбэртэй ХАРЬЦАНГУЙ зам (жишээ:
// `/api/files/acquisition/<id>/<uuid>_file.pdf`). Frontend-ийн proxy үүнийг
// MinIO руу дамжуулдаг тул орчин (host/port) солигдоход DB-д хөндөх шаардлагагүй.
type S3Storage struct {
	client *minio.Client
	// presignClient — browser хандах endpoint-оор гарын үсэг зурах client.
	// Presigned URL нь host-ыг гарын үсэгт оруулдаг тул API-ийн дотоод хаяг
	// (жишээ: minio:9000) browser-т ажиллахгүй — нийтийн хаягаар зурна.
	presignClient *minio.Client
	bucket        string
	baseURL       string
	// objectBase — GeoServer растрыг унших ШУУД хаягийн угтвар
	// (жишээ: http://minio:9000)
	objectBase string
}

// S3Options — хадгалалтад ГУРВАН өөр өнцгөөс хандах хаягийг тодорхойлно.
// Эдгээр нь орчноос шалтгаалж зөрдөг тул тусад нь тохируулагдана.
type S3Options struct {
	// Endpoint — API өөрөө хандах хаяг (host:port)
	Endpoint string
	// PublicEndpoint — BROWSER хандах хаяг (presigned байршуулалт). Хоосон бол
	// Endpoint.
	PublicEndpoint string
	// GeoServerEndpoint — GEOSERVER хандах хаяг (растрыг HTTP Range-ээр унших).
	// Хоосон бол Endpoint.
	GeoServerEndpoint string
	AccessKey         string
	SecretKey         string
	Bucket            string
	// BaseURL — DB-д бичигдэх ХАРЬЦАНГУЙ угтвар (/api/files)
	BaseURL string
	UseSSL  bool
}

func NewS3Storage(opts S3Options) (*S3Storage, error) {
	if opts.Endpoint == "" || opts.Bucket == "" {
		return nil, errors.New("s3 endpoint and bucket are required")
	}
	creds := credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, "")
	client, err := minio.New(opts.Endpoint, &minio.Options{Creds: creds, Secure: opts.UseSSL})
	if err != nil {
		return nil, err
	}

	presignClient := client
	if opts.PublicEndpoint != "" {
		host, ssl := splitEndpoint(opts.PublicEndpoint, opts.UseSSL)
		if host != opts.Endpoint || ssl != opts.UseSSL {
			presignClient, err = minio.New(host, &minio.Options{Creds: creds, Secure: ssl})
			if err != nil {
				return nil, fmt.Errorf("s3 public endpoint invalid: %w", err)
			}
		}
	}

	objectBase := strings.TrimRight(client.EndpointURL().String(), "/")
	if opts.GeoServerEndpoint != "" {
		host, ssl := splitEndpoint(opts.GeoServerEndpoint, opts.UseSSL)
		scheme := "http"
		if ssl {
			scheme = "https"
		}
		objectBase = scheme + "://" + host
	}

	return &S3Storage{
		client:        client,
		presignClient: presignClient,
		bucket:        opts.Bucket,
		baseURL:       strings.TrimRight(opts.BaseURL, "/"),
		objectBase:    objectBase,
	}, nil
}

// splitEndpoint — "http://host:9000" → ("host:9000", false). Схем байхгүй бол
// өгөгдсөн default-ыг хэрэглэнэ.
func splitEndpoint(endpoint string, defaultSSL bool) (host string, useSSL bool) {
	switch {
	case strings.HasPrefix(endpoint, "https://"):
		return strings.TrimRight(strings.TrimPrefix(endpoint, "https://"), "/"), true
	case strings.HasPrefix(endpoint, "http://"):
		return strings.TrimRight(strings.TrimPrefix(endpoint, "http://"), "/"), false
	default:
		return strings.TrimRight(endpoint, "/"), defaultSSL
	}
}

// EnsureBucket — bucket байхгүй бол үүсгэж, зөвхөн УНШИХ (download) нийтийн
// бодлого тохируулна.
//
// Яагаад нийтийн унших: DB-д хадгалагдсан file_url нь browser-ийн <a href>-д
// шууд ашиглагддаг. href нь Authorization header дамжуулж чаддаггүй тул
// объектыг унших нь токен шаарддаг байвал бүх татах линк эвдэрнэ. (Өмнөх
// `/mock-files` static хадгалалт мөн адил танилтгүй байсан — регресс биш.)
// Бичих/устгах нь зөвхөн API-ийн эрхтэй хийгдэнэ; жагсаах эрх нээгээгүй.
func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("bucket check failed: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("bucket create failed: %w", err)
		}
		slog.Info("file storage bucket created", "bucket", s.bucket)
	}

	policy := `{
	  "Version": "2012-10-17",
	  "Statement": [{
	    "Effect": "Allow",
	    "Principal": {"AWS": ["*"]},
	    "Action": ["s3:GetObject"],
	    "Resource": ["arn:aws:s3:::` + s.bucket + `/*"]
	  }]
	}`
	if err := s.client.SetBucketPolicy(ctx, s.bucket, policy); err != nil {
		return fmt.Errorf("bucket policy failed: %w", err)
	}
	return nil
}

// objectKey — хадгалалтын дотоод зам. namespace/ownerID/filename бүтэц нь
// хуучин локал хадгалалттай ижил тул шилжүүлэлт шулуун байна.
func objectKey(namespace, ownerID, storedName string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{namespace, ownerID, storedName} {
		if p = strings.Trim(path.Clean("/"+p), "/"); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

func (s *S3Storage) Save(ctx context.Context, namespace, ownerID, filename, contentType string, data []byte, unique bool) (*StoredFile, error) {
	return s.SaveStream(ctx, namespace, ownerID, filename, contentType, bytes.NewReader(data), int64(len(data)), unique)
}

// SaveStream — объектыг урсгалаар хуулна. Хэмжээг мэдэж байвал minio-go тохирох
// part хэмжээгээр илгээх тул санах ойн зарцуулалт файлын хэмжээнээс хамаарахгүй.
// size<0 бол хэмжээ тодорхойгүй гэж үзнэ.
func (s *S3Storage) SaveStream(ctx context.Context, namespace, ownerID, filename, contentType string, r io.Reader, size int64, unique bool) (*StoredFile, error) {
	cleanName, storedName := storedFileName(filename, unique)
	key := objectKey(namespace, ownerID, storedName)

	if contentType == "" {
		// Өргөтгөлөөс тодорхойлно — browser PDF/зургийг татахын оронд нээж
		// харуулах боломжтой болно (octet-stream бол үргэлж татагдана).
		contentType = mime.TypeByExtension(strings.ToLower(path.Ext(cleanName)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	info, err := s.client.PutObject(ctx, s.bucket, key, r, size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	return &StoredFile{
		URL:       s.urlForKey(key),
		Name:      cleanName,
		Type:      contentType,
		SizeBytes: info.Size,
	}, nil
}

func (s *S3Storage) Read(ctx context.Context, fileURL string) ([]byte, error) {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// PresignUpload — browser шууд PUT хийх зөвшөөрөл гаргана.
//
// Гарын үсэг зөвхөн method/host/key/хугацааг хамардаг (payload нь unsigned)
// тул browser Content-Type-аа чөлөөтэй явуулж болно. Object key нь
// namespace/ownerID-аар хязгаарлагдсан — дуудагч өөр хавтас руу бичиж чадахгүй.
func (s *S3Storage) PresignUpload(ctx context.Context, namespace, ownerID, filename, contentType string, expiry time.Duration) (*PresignedUpload, error) {
	_, storedName := storedFileName(filename, false)
	key := objectKey(namespace, ownerID, storedName)

	u, err := s.presignClient.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("presign failed: %w", err)
	}
	return &PresignedUpload{
		URL:              u.String(),
		Method:           http.MethodPut,
		FileURL:          s.urlForKey(key),
		StoredName:       storedName,
		ExpiresInSeconds: int(expiry.Seconds()),
	}, nil
}

func (s *S3Storage) URLFor(namespace, ownerID, storedName string) string {
	return s.urlForKey(objectKey(namespace, ownerID, storedName))
}

func (s *S3Storage) ObjectURL(fileURL string) (string, error) {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		return "", err
	}
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return s.objectBase + "/" + s.bucket + "/" + strings.Join(segments, "/"), nil
}

func (s *S3Storage) Stat(ctx context.Context, fileURL string) (*ObjectStat, error) {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		return nil, err
	}
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &ObjectStat{SizeBytes: info.Size, ContentType: info.ContentType}, nil
}

func (s *S3Storage) ReadRange(ctx context.Context, fileURL string, offset, length int64) ([]byte, error) {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		return nil, err
	}
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(offset, offset+length-1); err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, opts)
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	buf := make([]byte, length)
	n, err := io.ReadFull(obj, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

func (s *S3Storage) OpenStream(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		return nil, err
	}
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *S3Storage) Delete(ctx context.Context, fileURL string) error {
	key, err := s.keyFromURL(fileURL)
	if err != nil {
		// Танихгүй URL (жишээ: шилжүүлээгүй хуучин бичлэг) — устгах гэж
		// хөөцөлдөхгүй, харин дуудагчийн урсгалыг тасалдуулахгүй.
		slog.Warn("file delete skipped: unrecognized URL", "url", fileURL, "error", err)
		return nil
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3Storage) urlForKey(key string) string {
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return s.baseURL + "/" + strings.Join(segments, "/")
}

// keyFromURL — file_url-ээс object key-г гаргана.
func (s *S3Storage) keyFromURL(fileURL string) (string, error) {
	rel, ok := strings.CutPrefix(fileURL, s.baseURL+"/")
	if !ok {
		// Абсолют URL байж болно (өөр орчны host) — зөвхөн path-ыг тулгана.
		parsed, err := url.Parse(fileURL)
		if err != nil {
			return "", err
		}
		rel, ok = strings.CutPrefix(parsed.Path, s.baseURL+"/")
		if !ok {
			return "", fmt.Errorf("file URL is outside file storage: %s", fileURL)
		}
	}
	if decoded, err := url.PathUnescape(rel); err == nil {
		rel = decoded
	}
	if rel == "" || strings.Contains(rel, "..") {
		return "", errors.New("invalid file URL")
	}
	return rel, nil
}
