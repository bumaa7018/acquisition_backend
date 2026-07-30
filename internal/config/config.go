package config

import (
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	App             AppConfig
	DB              DBConfig `envPrefix:"DB_"`
	AuthDB          DBConfig `envPrefix:"AUTH_DB_"`
	JWT             JWTConfig
	Admin           AdminConfig
	Employee        EmployeeConfig
	Finance         FinanceConfig
	ProfessionalOrg ProfessionalOrgConfig
	Email           EmailConfig
	FileServer      FileServerConfig
	GeoServer       GeoServerConfig
}

type AppConfig struct {
	Port     string `env:"APP_PORT" envDefault:"8080"`
	Env      string `env:"APP_ENV" envDefault:"development"`
	BaseURL  string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"warn"`
}

// DBConfig нь envPrefix-тэй хослон ашиглагдана.
// DB_* → үндсэн DB, AUTH_DB_* → auth DB
type DBConfig struct {
	Host     string `env:"HOST,required"`
	Port     string `env:"PORT" envDefault:"5432"`
	User     string `env:"USER,required"`
	Password string `env:"PASSWORD,required"`
	Name     string `env:"NAME,required"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable"`
}

type JWTConfig struct {
	AccessSecret  string        `env:"JWT_ACCESS_SECRET,required"`
	RefreshSecret string        `env:"JWT_REFRESH_SECRET,required"`
	AccessTTL     time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTTL    time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`
}

type AdminConfig struct {
	Email    string `env:"ADMIN_EMAIL" envDefault:"admin@example.com"`
	Password string `env:"ADMIN_PASSWORD" envDefault:"Admin123!"`
}

type EmployeeConfig struct {
	Username string `env:"EMPLOYEE_USERNAME" envDefault:"employee"`
	Email    string `env:"EMPLOYEE_EMAIL" envDefault:"employee@example.com"`
	Password string `env:"EMPLOYEE_PASSWORD" envDefault:"Employee123!"`
}

// FinanceConfig — санхүүгийн мэргэжилтэн хэрэглэгчийн seed мэдээлэл.
type FinanceConfig struct {
	Username string `env:"FINANCE_USERNAME" envDefault:"finance"`
	Email    string `env:"FINANCE_EMAIL" envDefault:"finance@example.com"`
	Password string `env:"FINANCE_PASSWORD" envDefault:"Finance123!"`
}

// ProfessionalOrgConfig — мэргэжлийн байгууллагын хэрэглэгчийн seed мэдээлэл.
type ProfessionalOrgConfig struct {
	Username string `env:"PROF_ORG_USERNAME" envDefault:"prof_org"`
	Email    string `env:"PROF_ORG_EMAIL" envDefault:"prof_org@example.com"`
	Password string `env:"PROF_ORG_PASSWORD" envDefault:"ProfOrg123!"`
}

type EmailConfig struct {
	Host     string `env:"EMAIL_HOST" envDefault:""`
	Port     int    `env:"EMAIL_PORT" envDefault:"587"`
	Username string `env:"EMAIL_USERNAME" envDefault:""`
	Password string `env:"EMAIL_PASSWORD" envDefault:""`
	From     string `env:"EMAIL_FROM" envDefault:"noreply@example.com"`
	ResetURL string `env:"EMAIL_RESET_URL" envDefault:"http://localhost:3000/reset-password?token="`
}

// FileServerConfig — системд оруулсан БҮХ файлын хадгалалт.
//
// Mode="s3" (анхдагч) үед файл MinIO/S3 дээр хадгалагдаж, API нь өөр дээрээ
// файл БИЧИХГҮЙ — зөвхөн object-ийн замыг (file_url) DB-д тэмдэглэнэ.
// Mode="mock" нь хуучин локал хадгалалт — зөвхөн тест/шилжилтэд.
//
// BaseURL нь ХАРЬЦАНГУЙ зам (/api/files) — frontend-ийн proxy тэрийг MinIO руу
// дамжуулна. Ингэснээр DB-д бичигдсэн URL нь орчноос (host/port) хамаарахгүй.
type FileServerConfig struct {
	BaseURL  string `env:"FILE_SERVER_BASE_URL" envDefault:"/api/files"`
	LocalDir string `env:"FILE_SERVER_LOCAL_DIR" envDefault:"uploads/mock-file-server"`
	Mode     string `env:"FILE_SERVER_MODE" envDefault:"s3"`

	S3Endpoint string `env:"S3_ENDPOINT" envDefault:"localhost:9000"`
	// S3GeoServerEndpoint — GEOSERVER хандах хаяг. Дроны растрыг GeoServer
	// объектын хадгалалтаас HTTP Range хүсэлтээр ШУУД уншдаг тул түүний
	// хүрч чадах хаяг байх ёстой.
	//
	// Container-т API ба GeoServer нэг сүлжээнд байдаг тул анхдагчаар
	// S3_ENDPOINT-той ижил (minio:9000) — тохируулах шаардлагагүй. Харин
	// API-г HOST дээр (air) ажиллуулах үед S3_ENDPOINT=localhost:9000 болох ба
	// GeoServer container-аас "localhost" нь ӨӨРИЙГӨӨ заах тул ЭНД дотоод
	// сүлжээний хаягийг зааж өгнө: http://minio:9000
	S3GeoServerEndpoint string `env:"S3_GEOSERVER_ENDPOINT"`
	S3AccessKey         string `env:"S3_ACCESS_KEY" envDefault:"minioadmin"`
	S3SecretKey         string `env:"S3_SECRET_KEY" envDefault:"minioadmin"`
	S3Bucket            string `env:"S3_BUCKET" envDefault:"gov-files"`
	S3UseSSL            bool   `env:"S3_USE_SSL" envDefault:"false"`
}

// GeoServerConfig — дроны ортофотог давхарга болгон нийтлэхэд хэрэглэнэ.
//
// Растрын ЭХ СУРВАЛЖ нь файлын систем (MinIO): GeoServer объектыг HTTP Range
// хүсэлтээр ШУУД уншина (COG). Иймд хуваалцсан хавтас, локал зам шаардахгүй —
// backend растрыг өөр дээрээ хадгалахгүй.
//
// Workspace нь аль хэдийн байх ёстой (government-geoserver: make config).
type GeoServerConfig struct {
	URL       string `env:"GEOSERVER_URL" envDefault:"http://localhost:8600/geoserver"`
	User      string `env:"GEOSERVER_ADMIN" envDefault:"admin"`
	Password  string `env:"GEOSERVER_PASS" envDefault:"geoserver"`
	Workspace string `env:"GEOSERVER_WORKSPACE" envDefault:"land"`
}

func (e EmailConfig) Enabled() bool {
	return e.Host != "" && e.Username != ""
}

func (d DBConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + d.Port +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" sslmode=" + d.SSLMode
}

func (d DBConfig) MigrateURL() string {
	return "postgres://" + d.User + ":" + d.Password +
		"@" + d.Host + ":" + d.Port + "/" + d.Name +
		"?sslmode=" + d.SSLMode
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	slog.Info("config loaded",
		"env", cfg.App.Env,
		"port", cfg.App.Port,
		"main_db", cfg.DB.Name,
		"auth_db", cfg.AuthDB.Name,
		"email_enabled", cfg.Email.Enabled(),
		"log_level", cfg.App.LogLevel,
	)
	return cfg, nil
}

// ParseLogLevel .env-ийн LOG_LEVEL утгыг slog.Level рүү хөрвүүлнэ (debug|info|warn|error), таарахгүй бол info.
func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
