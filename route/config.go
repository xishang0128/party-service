package route

import (
	"net/http"
	"sparkle-service/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Config struct {
	CoreName   string `json:"core-name"`
	CoreDir    string `json:"core-dir"`
	ConfigPath string `json:"config-path"`
	WorkDir    string `json:"workdir"`
	LogPath    string `json:"log-path"`
	Secret     string `json:"secret"`
	Http       string `json:"http-listen"`
	NamedPipe  string `json:"named-pipe"`
	UnixSocket string `json:"unix-socket"`
}

func configRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getConfigs)
	r.Get("/{name}", getConfig)
	r.Post("/", updateConfig)
	return r
}

func getConfigs(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, config.GetConfig())
}

func getConfig(w http.ResponseWriter, r *http.Request) {
	var s string
	switch chi.URLParam(r, "name") {
	case "core-name":
		s = config.GetCoreName()
	case "core-dir":
		s = config.GetCoreDir()
	case "config-path":
		s = config.GetConfigPath()
	case "workdir":
		s = config.GetWorkDir()
	case "log-path":
		s = config.GetLogPath()
	default:
		http.Error(w, "Invalid config name", http.StatusBadRequest)
		return
	}

	render.JSON(w, r, s)
}

func updateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := render.DecodeJSON(r.Body, &cfg); err != nil {
		sendError(w, err)
		return
	}
	if err := config.UpdateConfig(cfg.CoreName, cfg.CoreDir, cfg.ConfigPath, cfg.WorkDir, cfg.LogPath, cfg.Secret, cfg.Http, cfg.NamedPipe, cfg.UnixSocket); err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, "success")
}
