package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string

	// is alive will return true, if server is alive and able to serve request
	IsAlive() bool

	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)

	// check if error
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

type LoadBalancer struct {
	port            string
	rountRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		rountRobinCount: 0,
		servers:         servers,
	}
}

// getNextServerAddr returns the address of the next available server to send a
// request to, using a simple round-robin algorithm
func (lb *LoadBalancer) getNextAvaiableServer() Server {
	server := lb.servers[lb.rountRobinCount%len(lb.servers)]

	for !server.IsAlive() {
		lb.rountRobinCount++
		server = lb.servers[lb.rountRobinCount%len(lb.servers)]
	}

	lb.rountRobinCount++

	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvaiableServer()
	fmt.Printf("Forwarding Request to Address %q\n", targetServer.Address())

	// could delete pre-existing X-Forwarded-For header to prevent IP spoofing
	targetServer.Serve(rw, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.google.com"),
	}

	lb := NewLoadBalancer("8081", servers)

	handleRedirect := func(rw http.ResponseWriter, request *http.Request) {
		lb.serveProxy(rw, request)
	}

	// register a proxy handler to handle all requests
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Serving request at 'localhost: %s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
