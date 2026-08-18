package service

import (
	"context"
	"io"
)

// SourceFile — эх системийн файл серверээс нээсэн файлын урсгал.
// Дуудагч Content-ийг ХААХ хариуцлагатай.
type SourceFile struct {
	ContentType string
	SizeBytes   int64
	Content     io.ReadCloser
}

// SourceFileClient — ЭХ СИСТЕМИЙН (ГУС) файл сервер: ЗӨВХӨН УНШИНА.
//
// Системийн ҮНДСЭН файл сервер нь MinIO (filestorage.Storage) — гараар
// оруулсан бүх файл тэнд хадгалагдана. Энэ интерфейс нь ГАДНЫ системийн
// FTP сервертэй холбогдож мэдээлэл ТАТАХ зориулалттай: бичих/устгах үйлдэл
// БАЙХГҮЙ.
//
// Хавсралтын файл нь дундын сервисийн DB-д БАЙХГҮЙ — зөвхөн зам (file_url) ба
// аль сервер дээр байгаа нь (ftp_id) бүртгэлтэй. Файлыг browser руу шууд
// өгөхгүй: сервер дотоод сүлжээнд байж болох ба хандалтын эрх шалгагдах ёстой
// тул API нь эндээс уншиж дамжуулна.
type SourceFileClient interface {
	// Open — ftpID дээрх filePath файлыг нээнэ.
	Open(ctx context.Context, ftpID, filePath string) (*SourceFile, error)
}
