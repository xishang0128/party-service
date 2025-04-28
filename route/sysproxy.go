package route

import (
	"fmt"
	"net/http"
	"sparkle-service/manager"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Request struct {
	Desktop string `json:"desktop"`
	UID     uint32 `json:"uid"`
	Server  string `json:"server"`
	Bypass  string `json:"bypass"`
	Url     string `json:"url"`
}

func httpProxyRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/*", status)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/disable", disable)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	uid, err := strconv.ParseUint(r.Header.Get("UID"), 10, 32)
	if err != nil {
		sendError(w, fmt.Errorf("invalid UID: %v", err))
		return
	}
	status, err := manager.QueryProxySettings(chi.URLParam(r, "*"), uint32(uid))
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func pac(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	err := manager.SetPac(req.Url, req.Desktop, req.UID)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	err := manager.SetProxy(req.Server, req.Bypass, req.Desktop, req.UID)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func disable(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	err := manager.DisableProxy(req.Desktop, req.UID)
	if err != nil {
		sendError(w, err)
	}
	render.NoContent(w, r)
}

func decodeRequest(r *http.Request, v any) error {
	if r.ContentLength > 0 {
		return render.DecodeJSON(r.Body, v)
	}
	return nil
}
