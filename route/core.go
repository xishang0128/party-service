package route

import (
	"net/http"

	"party-service/manager"

	"github.com/go-chi/chi/v5"
)

func coreManager() http.Handler {
	r := chi.NewRouter()
	r.Get("/", coreStatus)
	r.Post("/start", coreStart)
	r.Post("/stop", coreStop)
	r.Post("/restart", coreRestart)
	r.Post("/test", coreTest)
	return r
}

func coreStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("status"))
}

func coreStart(w http.ResponseWriter, r *http.Request) {
	err := manager.StartCore()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func coreStop(w http.ResponseWriter, r *http.Request) {
	err := manager.StopCore()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func coreRestart(w http.ResponseWriter, r *http.Request) {
	err := manager.RestartCore()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func coreTest(w http.ResponseWriter, r *http.Request) {
	err := manager.ConfigTest()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("test success\n"))
}
