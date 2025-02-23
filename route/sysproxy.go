package route

import (
	"net/http"
	"party-service/manager"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func httpProxyRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", status)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/disable", disable)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	status, err := manager.QueryProxySettings()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, err.Error())
		return
	}
	render.JSON(w, r, status)
}

func pac(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Server string `json:"server"`
	}{}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}

	err := manager.SetPac(req.Server)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, err.Error())
		return
	}
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Server string `json:"server"`
		Bypass string `json:"bypass"`
	}{}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}

	err := manager.SetProxy(req.Server, req.Bypass)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, err.Error())
		return
	}
	render.NoContent(w, r)
}

func disable(w http.ResponseWriter, r *http.Request) {
	err := manager.DisableProxy()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, err.Error())
		return
	}
	render.NoContent(w, r)
}
