package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// ===== Proxy Transport =====
type authTransport struct {
	base     http.RoundTripper
	endpoint Endpoint
	authMgr  *AuthManager
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch t.endpoint.AuthType {
	case "bearer":
		token, err := t.authMgr.GetToken(t.endpoint)
		if err != nil {
			log.Printf("Auth error for %s: %v", t.endpoint.Name, err)
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case "token":
		token, err := t.authMgr.GetToken(t.endpoint)
		if err != nil {
			log.Printf("Auth error for %s: %v", t.endpoint.Name, err)
			return nil, err
		}
		req.Header.Set("Authorization", "Token "+token)
	}
	return t.base.RoundTrip(req)
}

// ===== Reverse Proxy Builder =====
func newProxy(e Endpoint, authMgr *AuthManager) *httputil.ReverseProxy {
	target, err := url.Parse(e.TargetURL)
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &authTransport{base: http.DefaultTransport, endpoint: e, authMgr: authMgr}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/"+e.Name)
	}
	return proxy
}

// ===== Restricted Proxy Handler =====
func newRestrictedProxy(e Endpoint, authMgr *AuthManager) http.Handler {
	proxy := newProxy(e, authMgr)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/"+e.Name)
		log.Printf("Request to %s:%s", e.Name, path)
		allowed := false
		for _, a := range e.Allowed {
			if strings.HasPrefix(path, a.Path) {
				for _, m := range a.Methods {
					if strings.EqualFold(r.Method, m) {
						allowed = true
						break
					}
				}
				if allowed {
					break
				}
			}
		}

		if !allowed {
			http.Error(w, "Forbidden: endpoint or method not allowed", http.StatusForbidden)
			return
		}

		proxy.ServeHTTP(w, r)
	})
}

// ===== Main =====
func main() {
	configFile := flag.String("config", "config.yaml", "Path to config file (YAML or JSON)")
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	secretKey := os.Getenv("MASTER_KEY")
	if len(secretKey) != 32 {
		log.Fatal("MASTER_KEY must be set to 32 characters (AES-256 key)")
	}

	cfg, err := LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	authMgr := NewAuthManager(secretKey)

	for _, e := range cfg.Endpoints {
		prefix := "/api/" + e.Name
		http.Handle(prefix+"/", newRestrictedProxy(e, authMgr))
		log.Printf("Mapped %s -> %s (allowed paths=%v)", prefix, e.TargetURL, e.Allowed)
	}

	log.Printf("Proxy server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
