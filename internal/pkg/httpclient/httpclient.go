// Package httpclient нь бусад системрүү (гадаад API) HTTP хүсэлт илгээх нэгдсэн
// цэг. Ирээдүйд гадаад системтэй холбогдох ШИНЭ код ЭНЭ Client-ээр л дамжих
// ёстой — ингэснээр ямар ч дуудлага client_request хүснэгтэд бүртгэгдэхгүй
// үлдэхгүй (харилцан адилгүй газар бие даан http.Client үүсгэхээс зайлсхийнэ).
package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"acquisition_backend/internal/service"
)

// maxLoggedBodyBytes — request/response биеийг лог/DB-д хэт томроохоос
// сэргийлж энэ хэмжээгээр огтолж хадгална.
const maxLoggedBodyBytes = 16 * 1024

// maxBodyBytes — хүсэлт/хариуны биеийг бүхэлд нь санах ойд уншихдаа тавих
// хатуу дээд хязгаар. Энэ Client нь ердийн JSON/API дуудлагад зориулагдсан
// (хэдэн KB-KB хэмжээтэй) — том файл татах/илгээх шаардлагатай интеграцид
// зориулж дуудагч тал өөрийн, ЭНЭ хязгаараас чөлөөтэй http.Client ашиглах
// ёстой. Гадаад систем санамсаргүй (эсвэл санаатай) асар том хариу буцаавал
// process-ийн санах ойг барж авахаас сэргийлнэ.
const maxBodyBytes = 20 * 1024 * 1024

// readBodyCapped биеийг maxBodyBytes хүртэл уншиж, хэтэрвэл алдаа буцаана
// (чимээгүй огтлохгүй — дуудагч тал бүрэн бус өгөгдлийг андуурч ашиглахаас сэргийлнэ).
func readBodyCapped(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, maxBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBodyBytes {
		return nil, fmt.Errorf("хүсэлт/хариуны бие %dMB хязгаараас хэтэрсэн", maxBodyBytes/(1024*1024))
	}
	return data, nil
}

type Client struct {
	http *http.Client
	log  service.ClientRequestService
}

// New — log нь nil байж болохгүй (бүртгэлгүй гадаад дуудлага хийхийг зөвшөөрөхгүй).
func New(log service.ClientRequestService) *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
		log:  log,
	}
}

// Do нь бэлтгэсэн *http.Request-ийг гүйцэтгэж, амжилттай эсэхээс үл хамааран
// method/host/path/request-response/status/хугацааг client_request хүснэгтэд
// хадгална. Хариултын биеийг дуудагч тал дахин уншиж болохоор сэргээнэ.
//
// АНХААР: request/response бие бүхэлдээ хадгалагдана (шаардлагаар). Хэрэв
// ирээдүйд нууц мэдээлэл (API key, карт мэдээлэл г.м.) агуулсан систем рүү
// дуудлага хийвэл дуудагч тал биед нь нууц зүйлийг оруулахгүй байх, эсвэл
// дуудахаасаа өмнө биеэ цэвэрлэх хариуцлагатай.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Body != nil {
		body, err := readBodyCapped(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("хүсэлтийн бие уншихад алдаа гарлаа: %w", err)
		}
		reqBody = body
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	duration := time.Since(start)

	entry := service.ClientRequestEntry{
		Method:      req.Method,
		Host:        req.URL.Host,
		Path:        req.URL.Path,
		RequestBody: truncate(string(reqBody)),
		DurationMs:  duration.Milliseconds(),
		RequestedAt: start,
	}

	if err != nil {
		entry.Error = err.Error()
		c.log.Log(entry)
		return nil, err
	}

	var respBody []byte
	if resp.Body != nil {
		body, readErr := readBodyCapped(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			entry.StatusCode = resp.StatusCode
			entry.Error = readErr.Error()
			c.log.Log(entry)
			return nil, fmt.Errorf("хариуны бие уншихад алдаа гарлаа: %w", readErr)
		}
		respBody = body
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
	}
	entry.StatusCode = resp.StatusCode
	entry.RequestResponse = truncate(string(respBody))
	c.log.Log(entry)

	return resp, nil
}

func truncate(s string) string {
	if len(s) <= maxLoggedBodyBytes {
		return s
	}
	return s[:maxLoggedBodyBytes] + "...[truncated]"
}
