// Package logstack нь Loki+Promtail+Grafana лог хянах stack-ийг backend
// асах бүрд шалгаж, шаардлагатай бол автоматаар асаадаг.
package logstack

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	composeMain    = "docker-compose.yml"
	composeLogging = "docker-compose.logging.yml"
)

var requiredContainers = []string{"gov_loki", "gov_promtail", "gov_grafana"}

// EnsureRunning Loki/Promtail/Grafana ажиллаж байгаа эсэхийг шалгаад,
// ажиллахгүй бол docker compose-оор татаж асаана. Хэзээ ч сервэрийн
// асаалтыг зогсоохгүй (сая fatal биш) — зөвхөн best-effort:
//   - docker CLI байхгүй бол (жишээ нь production api container дотор,
//     docker socket холбогдоогүй) чимээгүй анхааруулга логлоод буцна.
//   - docker-compose файл олдохгүй (deployments/ фолдер image-д ороогүй)
//     тохиолдолд мөн чимээгүй буцна.
//
// rootDir нь project root (deployments/ дэд фолдертой) — dev орчинд air-г
// repo root-оос ажиллуулдаг тул "." хангалттай.
func EnsureRunning(ctx context.Context, rootDir string) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		slog.Warn("logging stack: docker cli олдсонгүй, шалгалтыг алгаслаа")
		return
	}

	deploymentsDir := filepath.Join(rootDir, "deployments")
	mainFile := filepath.Join(deploymentsDir, composeMain)
	loggingFile := filepath.Join(deploymentsDir, composeLogging)
	envFile := filepath.Join(rootDir, ".env")

	if !fileExists(mainFile) || !fileExists(loggingFile) {
		slog.Warn("logging stack: docker-compose файл олдсонгүй, шалгалтыг алгаслаа", "dir", deploymentsDir)
		return
	}

	if allRunning(ctx, dockerPath) {
		slog.Info("logging stack: Loki/Promtail/Grafana аль хэдийн ажиллаж байна")
		return
	}

	slog.Info("logging stack: Loki/Promtail/Grafana ажиллахгүй байгаа тул асааж байна...")
	args := []string{"compose", "-f", mainFile, "-f", loggingFile}
	if fileExists(envFile) {
		// deploy.sh-тай адил root .env-ийг ашиглана (жишээ нь GRAFANA_ADMIN_PASSWORD).
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "up", "-d", "--pull", "missing", "loki", "promtail", "grafana")
	cmd := exec.CommandContext(ctx, dockerPath, args...)
	cmd.Dir = deploymentsDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		slog.Warn("logging stack: асаахад алдаа гарлаа", "error", err, "stderr", stderr.String())
		return
	}
	slog.Info("logging stack: Loki/Promtail/Grafana амжилттай асаалаа", "grafana", "http://localhost:3001")
}

func allRunning(ctx context.Context, dockerPath string) bool {
	for _, name := range requiredContainers {
		cmd := exec.CommandContext(ctx, dockerPath, "inspect", "-f", "{{.State.Running}}", name)
		out, err := cmd.Output()
		if err != nil || string(bytes.TrimSpace(out)) != "true" {
			return false
		}
	}
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
