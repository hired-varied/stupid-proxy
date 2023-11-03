package main

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/hired-varied/stupid-proxy/utils"
	"github.com/pires/go-proxyproto"
)

var (
	configFile = flag.String("config-file", "./config.yaml", "Config file")
	logger     = utils.NewLogger(utils.InfoLevel)
)

// Config represents the server configuration.
type Config struct {
	UpstreamAddr    string            `yaml:"upstream_addr"`
	ListenAddr      string            `yaml:"listen_addr"`
	AuthTriggerPath string            `yaml:"auth_trigger_path"`
	Auth            map[string]string `yaml:"auth"`
}

type defaultHandler struct {
	reverseProxy *httputil.ReverseProxy
	config       Config
}

type flushWriter struct {
	w io.Writer
}

func (f *flushWriter) Write(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = errors.New(s)
				logger.Error("Flush writer error in recover: %s\n", err)
				return
			}
			err = r.(error)
		}
	}()

	n, err = f.w.Write(p)
	if err != nil {
		logger.Error("Flush writer error in write response: %s\n", err)
		return
	}
	if f, ok := f.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}

var headerBlackList = map[string]bool{}

func (h *defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isAuthTriggerURL := r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, h.config.AuthTriggerPath)
	authorized, username := h.isAuthenticated(r.Header.Get("Proxy-Authorization"))
	if isAuthTriggerURL {
		if authorized {
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Add("Proxy-Authenticate", "Basic realm=\"Hi, please show me your token!\"")
			w.WriteHeader(http.StatusProxyAuthRequired)
		}
		w.Write([]byte(""))
		w.(http.Flusher).Flush()
	} else {
		if authorized {
			logger.Debug("[%s] %s %s\n", username, r.Method, r.URL)
			for k := range r.Header {
				if headerBlackList[strings.ToLower(k)] {
					r.Header.Del(k)
				}
			}
			proxy(w, r, username)
		} else { // silent makes big fortune
			if username == "" {
				logger.Debug("[normal] %s %s\n", r.Method, r.URL)
			} else {
				logger.Debug("{%s} %s %s\n", username, r.Method, r.URL)
			}
			h.handleReverseProxy(w, r)
		}
	}
}

func (h *defaultHandler) isAuthenticated(authHeader string) (bool, string) {
	s := strings.SplitN(authHeader, " ", 2)
	if len(s) != 2 {
		return false, ""
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false, "AuthBase64Invalid"
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false, "AuthUsernamePasswordInvalid"
	}

	email := pair[0]
	token := pair[1]

	// Check if it matches the static result
	if h.config.Auth[email] == token {
		return true, email
	}

	return false, "InvalidEmail " + email
}

func proxy(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method == http.MethodConnect {
		handleTunneling(w, r, username)
	} else {
		handleHTTP(w, r, username)
	}
}

func (h *defaultHandler) handleReverseProxy(w http.ResponseWriter, r *http.Request) {
	h.reverseProxy.ServeHTTP(w, r)
}

func createTCPConn(host string) (*net.TCPConn, error) {
	destConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if tcpConn, ok := destConn.(*net.TCPConn); ok {
		return tcpConn, nil
	}
	return nil, errors.New("failed to cast net.Conn to net.TCPConn")
}

func hijack(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijacking not supported")
	}
	clientConn, _, err := hijacker.Hijack()
	return clientConn, err
}

func handleTunneling(w http.ResponseWriter, r *http.Request, username string) {
	remoteTCPConn, err := createTCPConn(r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer remoteTCPConn.Close()
	w.WriteHeader(http.StatusOK)
	if r.ProtoMajor == 2 {
		w.(http.Flusher).Flush() // Must flush, or the client won't start the connection
		go func() {
			// Client -> Remote
			defer remoteTCPConn.CloseWrite()
			utils.CopyAndPrintError(remoteTCPConn, r.Body, logger)
		}()
		// Remote -> Client
		defer remoteTCPConn.CloseRead()
		utils.CopyAndPrintError(&flushWriter{w}, remoteTCPConn, logger)
	} else {
		clientConn, err := hijack(w)
		if err != nil {
			logger.Error("Hijack failed: %s", err)
			return
		}
		defer clientConn.Close()
		go func() {
			// Client -> Remote
			defer remoteTCPConn.CloseWrite()
			utils.CopyAndPrintError(remoteTCPConn, clientConn, logger)
		}()
		// Remote -> Client
		defer remoteTCPConn.CloseRead()
		utils.CopyAndPrintError(clientConn, remoteTCPConn, logger)
	}
}

func handleHTTP(w http.ResponseWriter, req *http.Request, username string) {
	if req.ProtoMajor == 2 {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
	}
	pipeRead, pipeWrite := io.Pipe()
	fromBody := req.Body
	req.Body = pipeRead
	go func() {
		defer pipeWrite.Close()
		defer fromBody.Close()
		utils.CopyAndPrintError(pipeWrite, fromBody, logger)
	}()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	utils.CopyAndPrintError(w, resp.Body, logger)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	flag.Parse()
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Trailer",
		"TE",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, header := range hopByHopHeaders {
		headerBlackList[strings.ToLower(header)] = true
	}
	config := &Config{}
	utils.LoadConfigFile(*configFile, config)
	reverseProxyURL, err := url.Parse(config.UpstreamAddr)
	if err != nil {
		log.Fatal("Failed to parse reverse proxy URL", err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(reverseProxyURL)
	logger.Info("Listening on %s, upstream to %s.\n", config.ListenAddr, config.UpstreamAddr)

	h2s := &http2.Server{}
	server := &http.Server{
		Addr: config.ListenAddr,
		Handler: h2c.NewHandler(&defaultHandler{
			reverseProxy,
			*config,
		}, h2s),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(err)
	}

	proxyListener := &proxyproto.Listener{
		Listener: ln,
	}
	ln = proxyListener
	defer ln.Close()

	err = server.Serve(ln)
	if err != nil {
		log.Fatal("Failed to serve: ", err)
	}
}
