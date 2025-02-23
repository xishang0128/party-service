package route

import (
	"io"
	"net/http"
	"strings"

	"party-service/data"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	ErrBadRequest   = render.M{"error": "Bad request"}
	ErrUnauthorized = render.M{"error": "Unauthorized"}
)

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Group(func(r chi.Router) {
		r.Use(auth())
		r.Get("/", hello)
		r.Mount("/data", dataRouter())
		r.Mount("/http_proxy", httpProxyRouter())
		r.Mount("/core", coreManager())
	})
	return r
}

func dataRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getData)

	r.Route("/{name}", func(r chi.Router) {
		r.Get("/", getData)
		r.Put("/", setData)
	})
	return r
}

func auth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			bearer, token, found := strings.Cut(r.Header.Get("Authorization"), " ")

			if bearer != "Bearer" || !found || !strings.EqualFold(token, secret) {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, "200 ok")
}

func getData(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	v := data.Data().Get(name)

	render.JSON(w, r, v)
}

func setData(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, render.M{"error": err.Error()})
		return
	}
	defer r.Body.Close()

	name := chi.URLParam(r, "name")

	data.Data().Set(name, string(body))

	render.JSON(w, r, "200 ok")
}
