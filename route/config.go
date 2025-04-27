package route

import (
	"net/http"
	"sparkle-service/config"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func configRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getConfigs)
	r.Get("/{name}", getConfig)
	r.Post("/{name}", updateConfig)
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
	var cfg struct {
		Value string `json:"value"`
	}
	if err := render.DecodeJSON(r.Body, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch chi.URLParam(r, "name") {
	case "core-name":
		config.SetCoreName(cfg.Value)
	case "core-dir":
		cfg.Value = strings.TrimSuffix(cfg.Value, "/")
		config.SetCorePath(cfg.Value)
	case "config-path":
		config.SetConfigPath(cfg.Value)
	case "workdir":
		cfg.Value = strings.TrimSuffix(cfg.Value, "/")
		config.SetWorkDir(cfg.Value)
	case "log-path":
		config.SetLogPath(cfg.Value)
	case "secret":
		config.SetSecret(cfg.Value)
	case "http":
		config.SetHttp(cfg.Value)
	case "named-pipe":
		config.SetNamedPipe(cfg.Value)
	case "unix-socket":
		config.SetUnixSocket(cfg.Value)
	default:
		http.Error(w, "Invalid config name", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
