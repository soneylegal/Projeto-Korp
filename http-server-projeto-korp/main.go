package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// projetoKorpResponse descreve o formato exigido pelo desafio.
// As tags `json:"..."` dizem ao encoder como cada campo deve
// aparecer no JSON de saída.
type projetoKorpResponse struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

// httpRequestsTotal cobre o requisito de "volume de requisições":
// um contador com label de path e status, no padrão que o
// Prometheus espera encontrar em /metrics.
var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total de requisições HTTP recebidas pelo http-server-projeto-korp",
	},
	[]string{"path", "status"},
)

// statusRecorder existe só pra capturar o status code que o handler
// escreveu, já que http.ResponseWriter não deixa ler isso depois.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// withMetrics envolve qualquer handler pra contar a requisição,
// usando o path e o status final da resposta como labels.
func withMetrics(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		httpRequestsTotal.WithLabelValues(path, strconv.Itoa(rec.status)).Inc()
	}
}

// projetoKorpHandler atende GET /projeto-korp.
// Todo handler em Go segue essa mesma assinatura fixa:
// recebe onde escrever a resposta (w) e a requisição recebida (r).
func projetoKorpHandler(w http.ResponseWriter, r *http.Request) {
	resp := projetoKorpResponse{
		Nome:    "Projeto Korp",
		Horario: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")

	// Encode pode falhar (raro aqui, mas é o padrão do Go: toda
	// operação que pode dar errado retorna um `error`, e a gente
	// checa explicitamente em vez de depender de exceções).
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("erro ao codificar resposta: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/projeto-korp", withMetrics("/projeto-korp", projetoKorpHandler))

	// /metrics é só mais uma rota no mesmo mux — o handler pronto
	// da lib do Prometheus que expõe tudo no formato esperado
	// (incluindo o httpRequestsTotal declarado acima).
	mux.Handle("/metrics", promhttp.Handler())

	const addr = ":8080"
	log.Printf("http-server-projeto-korp ouvindo em %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}
