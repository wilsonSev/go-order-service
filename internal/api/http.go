package api

import (
	"net/http"
	"strings"
	"github.com/wilsonSev/go-order-service/internal/storage"
)

type Server struct {
	repo *storage.OrderRepo // репа
	mux *http.ServeMux // маршрутизатор, URL - хендлер
}

func NewServer(repo *storage.OrderRepo) *Server {
	s := &Server{repo: repo, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /order/", s.handleGetOrder)
}

// Реализуем интерфейс Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Реализуем функцию-геттер
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	// /order/{order_uid}
	uid := strings.TrimPrefix(r.URL.Path, "/order/")
	// отдельно делаем проверку на пустой uid
	if uid == "" {
		http.Error(w, "missing order_uid", http.StatusBadRequest)
		return
	}

	raw, err := s.repo.GetByUID(r.Context(), uid)
	if err != nil {
		if err == storage.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json") // передаем в header информацию о том, что это шаблон
	w.WriteHeader(http.StatusOK) // задаем статус OK, так как ошибка не была найдена
	_, _ = w.Write(raw)
}