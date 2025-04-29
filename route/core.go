package route

import (
	"io"
	"net/http"
	"sparkle-service/manager"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	cm     *manager.CoreManager
	isInit atomic.Bool
)

func coreManager() http.Handler {
	if !isInit.Load() {
		cm = manager.NewCoreManager()
		isInit.Store(true)
	}

	r := chi.NewRouter()

	r.Use(requestLogger)

	r.Get("/", coreStatus)
	r.Post("/start", coreStart)
	r.Post("/stop", coreStop)
	r.Post("/restart", coreRestart)
	r.Post("/test", coreTest)

	return r
}

func coreStatus(w http.ResponseWriter, r *http.Request) {
	status, err := cm.GetProcessInfo()
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func coreStart(w http.ResponseWriter, r *http.Request) {
	if err := cm.StartCore(); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "核心启动成功")
}

func coreStop(w http.ResponseWriter, r *http.Request) {
	if err := cm.StopCore(); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "核心停止成功")
}

func coreRestart(w http.ResponseWriter, r *http.Request) {
	if err := cm.RestartCore(); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "核心重启成功")
}

func coreTest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, err)
		return
	}
	if err := manager.ConfigCheck(string(body)); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "测试成功完成")
}
