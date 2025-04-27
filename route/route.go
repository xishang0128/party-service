package route

import (
	"net/http"
	"strings"

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
		r.Mount("/config", configRouter())
		r.Mount("/http_proxy", httpProxyRouter())
		r.Mount("/core", coreManager())
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
