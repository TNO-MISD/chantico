package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"example.com/m/internal/graph"
	"example.com/m/internal/html"
	"example.com/m/internal/kubernetes"
)

type HTTPServer struct {
	server *http.Server
	port   int
}

func New(r *html.TemplateRenderer, k *kubernetes.KubernetesClient, port int) *HTTPServer {
	h := &Handler{
		renderer:   r,
		kubernetes: k,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.HomePage)
	mux.HandleFunc("GET /long-request", h.LongRequest)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	return &HTTPServer{
		server: server,
		port:   port,
	}
}

func (s HTTPServer) Run(cancel context.CancelFunc) {
	fmt.Println("Starting HTTP server on port:", s.port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Error in HTTP server")
		cancel()
	}
}

func (s HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

type Handler struct {
	renderer   *html.TemplateRenderer
	kubernetes *kubernetes.KubernetesClient
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {

	nodes, err := h.kubernetes.GetDataCenterResources()
	if err != nil {
		panic(err)
	}

	m := graph.GenerateMermaidString(nodes)
	h.renderer.RenderHomePage(w, html.HomePageData{Diagram: m})
}

func (h *Handler) LongRequest(w http.ResponseWriter, r *http.Request) {

	fmt.Println("Sleeping for 10 seconds...")
	time.Sleep(35 * time.Second)
	fmt.Println("Awake now!")
}
