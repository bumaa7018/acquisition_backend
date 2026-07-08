package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodySizeLimit — хүсэлтийн биеийн (body) хэмжээг хязгаарлана. multipart/form-data
// (файл байршуулах) хүсэлтэд хэрэглэхгүй — тэдгээр handler-ууд өөрсдийн илүү өндөр
// MaxBytesReader хязгаараа тогтоодог. JSON зэрэг бусад хүсэлтийн хэт том биеэс
// (санах ойн DoS) сэргийлнэ.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() == "multipart/form-data" {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
