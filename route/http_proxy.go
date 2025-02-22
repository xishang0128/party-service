package route

import (
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func httpProxyRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", status)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/off", off)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{"status": "ok"})
}

func pac(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}
	defer r.Body.Close()

	log.Println(string(body))
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	render.NoContent(w, r)
}

func off(w http.ResponseWriter, r *http.Request) {
	render.NoContent(w, r)
}
