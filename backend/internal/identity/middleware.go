package identity

import "net/http"

type Middleware struct{}

func (m Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ContextWithActor(r.Context(), Actor{ID: "anonymous"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
