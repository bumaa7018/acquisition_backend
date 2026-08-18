package config_test

import (
	"testing"
	"time"

	"acquisition_backend/internal/config"

	"github.com/stretchr/testify/assert"
)

// ГУС-ийн хүснэгттэй тохирсон эсэх:
//
//	ftp_id=3 host=192.168.x.x user/pass root_dir=/DATA port=22 passive=1 ssl=1 timeout=30
//
// port=22 тул протокол нь SFTP (SSH) байх ёстой.
func TestSourceFileConfig_MatchesSourceSystemTable(t *testing.T) {
	target, ok := config.SourceFileConfig{
		FtpID: "3", Host: "192.168.10.20", Username: "gis_reader", Password: "P@ss word",
	}.FTPTarget()

	assert.True(t, ok)
	assert.Equal(t, config.ProtocolSFTP, target.Protocol, "порт 22 → SFTP")
	assert.Equal(t, "192.168.10.20:22", target.Addr)
	assert.Equal(t, "3", target.FtpID)
	assert.Equal(t, "gis_reader", target.Username)
	assert.Equal(t, "P@ss word", target.Password)
	assert.Equal(t, "/DATA", target.RootDir, "сүүлийн / хасагдана")
	assert.True(t, target.Passive)
	assert.False(t, target.ImplicitTLS, "SFTP-д FTPS TLS хамааралгүй")
	assert.Equal(t, 30*time.Second, target.Timeout)
}

// Зайг цэвэрлэнэ (.env-д санамсаргүй орсон зай).
func TestSourceFileConfig_TrimsSpaces(t *testing.T) {
	target, ok := config.SourceFileConfig{
		FtpID: " 3 ", Host: " 192.168.10.20 ", Username: " gis_reader ",
	}.FTPTarget()
	assert.True(t, ok)
	assert.Equal(t, "192.168.10.20:22", target.Addr)
	assert.Equal(t, "3", target.FtpID)
	assert.Equal(t, "gis_reader", target.Username)
}

// Host хоосон бол тохируулаагүй.
func TestSourceFileConfig_Enabled(t *testing.T) {
	assert.False(t, config.SourceFileConfig{}.Enabled())
	assert.False(t, config.SourceFileConfig{FtpID: "3", Username: "u"}.Enabled(),
		"host-гүй бол зөвхөн ftp_id/нэрээр нээгдэх ёсгүй")
	assert.True(t, config.SourceFileConfig{Host: "192.168.10.20"}.Enabled())

	_, ok := config.SourceFileConfig{}.FTPTarget()
	assert.False(t, ok)
}
