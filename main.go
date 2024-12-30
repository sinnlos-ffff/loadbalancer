package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	URL   *url.URL
	Alive bool
	mux   sync.RWMutex
}

func (s *Server) SetAlive(alive bool) {
	s.mux.Lock()
	s.Alive = alive
	s.mux.Unlock()
}

func (s *Server) IsAlive() bool {
	s.mux.RLock()
	alive := s.Alive
	s.mux.RUnlock()
	return alive
}

type LoadBalancer struct {
	servers []*Server
	proxy   *httputil.ReverseProxy
	current int
	mutex   sync.Mutex
}

func NewLoadBalancer(serverURLs []string) *LoadBalancer {
	servers := make([]*Server, 0)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// This will be set later
		},
	}

	for _, serverURL := range serverURLs {
		url, err := url.Parse(serverURL)
		if err != nil {
			log.Fatal(err)
		}

		servers = append(servers, &Server{
			URL:   url,
			Alive: true,
		})
	}

	lb := &LoadBalancer{
		servers: servers,
		proxy:   proxy,
		current: 0,
	}

	lb.proxy.Director = lb.director

	return lb
}

// NextServer returns the next available server using round-robin
func (lb *LoadBalancer) NextServer() *Server {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Loop through servers to find the next available one
	for i := 0; i < len(lb.servers); i++ {
		lb.current = (lb.current + 1) % len(lb.servers)
		if lb.servers[lb.current].IsAlive() {
			return lb.servers[lb.current]
		}
	}
	return nil
}

func (lb *LoadBalancer) director(req *http.Request) {
	server := lb.NextServer()
	if server != nil {
		req.URL.Scheme = server.URL.Scheme
		req.URL.Host = server.URL.Host
		req.Host = server.URL.Host
		log.Printf("Routing request to: %s\n", server.URL)
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(lb.servers) == 0 {
		http.Error(w, "No servers available", http.StatusServiceUnavailable)
		return
	}

	lb.proxy.ServeHTTP(w, r)
}

// HealthCheck performs health checks on backend servers
func (lb *LoadBalancer) HealthCheck() {
	for _, server := range lb.servers {
		go func(s *Server) {
			for {
				resp, err := http.Get(s.URL.String())
				if err != nil {
					s.SetAlive(false)
					log.Printf("Server %s is down\n", s.URL)
				} else {
					resp.Body.Close()
					s.SetAlive(true)
					log.Printf("Server %s is up\n", s.URL)
				}
				// Check every 30 seconds
				time.Sleep(30 * time.Second)
			}
		}(server)
	}
}

func main() {
	// Define backend servers
	serverURLs := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	// Create load balancer
	lb := NewLoadBalancer(serverURLs)

	// Start health checking
	go lb.HealthCheck()

	// Start the load balancer
	server := &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	log.Printf("Load Balancer started at :%v\n", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
