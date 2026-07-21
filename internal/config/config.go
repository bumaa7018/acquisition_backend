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

type FileServerConfig struct {
	BaseURL  string `env:"FILE_SERVER_BASE_URL" envDefault:"http://localhost:8080/mock-files"`
	LocalDir string `env:"FILE_SERVER_LOCAL_DIR" envDefault:"uploads/mock-file-server"`
	Mode     string `env:"FILE_SERVER_MODE" envDefault:"mock"`
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
