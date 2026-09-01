package main

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/prometheus/common/config"
)

// Header names.
const (
	redfishURLHeaderName = "X-Redfish-Url"
	realIPHeaderName     = "X-Real-IP"
	authorization        = "Authorization"
)

// Mutex to read/write to targets map.
var (
	targetsMapMu = sync.RWMutex{}
)

type rpConfig struct {
	logger  *slog.Logger
	redfish *Redfish
}

// NewMultiHostReverseProxy returns a new instance of ReverseProxy that routes requests
// to multiple targets based on remote address of the request.
func NewMultiHostReverseProxy(c *rpConfig) (*httputil.ReverseProxy, error) {
	// Make a map of host addr to bmc url using config
	targets := make(map[string]*url.URL)

	for _, target := range c.redfish.Config.Targets {
		for _, ip := range target.HostAddrs {
			targets[ip] = target.URL
		}
	}

	// Create a HTTP roundtripper
	httpRoundTripper, err := config.NewRoundTripperFromConfig(c.redfish.Config.HTTPClientConfig, "redfish_proxy", config.WithUserAgent("ceems/redfish_proxy"))
	if err != nil {
		return nil, err
	}

	rewrite := func(req *httputil.ProxyRequest) {
		rewriteRequestURL(c.logger, req, targets)
	}

	// Create a custom error handler that returns invalid request on all errors
	errorHandler := func(rw http.ResponseWriter, req *http.Request, err error) {
		c.logger.Error("failed to proxy request", "err", err)

		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("failed to find redfish target"))
	}

	return &httputil.ReverseProxy{Rewrite: rewrite, Transport: httpRoundTripper, ErrorHandler: errorHandler}, nil
}

// rewriteRequestURL rewrites the request URL to point to the target.
//
// We attempt to find the correct target using following methods:
//
// - Check X-Redfish-Url header
// - Lookup RemoteAddr and find the target from map of provided targets
//
// Always X-Redfish-Url header is checked for BMC hostname and if not found,
// target URL is looked up from provided targets.
func rewriteRequestURL(logger *slog.Logger, preq *httputil.ProxyRequest, targets map[string]*url.URL) {
	var target *url.URL

	var remoteIPs []string

	var err error

	var ok bool

	// First get the remote address of the client
	remoteIPs = preq.In.Header[http.CanonicalHeaderKey(realIPHeaderName)]

	// Add remoteAddr only when not on testing
	ip, _, err := net.SplitHostPort(preq.In.RemoteAddr)
	if err == nil && os.Getenv("__IS_TESTING") == "" {
		remoteIPs = append(remoteIPs, ip)
	}

	// Check if target is already in map
	targetsMapMu.RLock()

	for _, ip := range remoteIPs {
		if target, ok = targets[ip]; ok {
			// Unlock map and go to rewrite_req
			targetsMapMu.RUnlock()

			goto rewrite_req
		}
	}

	targetsMapMu.RUnlock()

	// If target is not found in map, check header
	// Always use CanonicalHeaderKey as golang always canonicalize headers
	// internally
	if targetURL := preq.In.Header.Get(redfishURLHeaderName); targetURL != "" {
		target, err = url.Parse(targetURL)
		if err != nil {
			logger.Error("Fetched Redfish URL from headers is invalid", "err", err)

			return
		}

		// Add this to targets map
		targetsMapMu.Lock()

		for _, ip := range remoteIPs {
			targets[ip] = target
		}

		targetsMapMu.Unlock()

		goto rewrite_req
	} else {
		// If no matches found, log the found remote IPs and return
		logger.Error("Failed to find target", "remote_ips", strings.Join(remoteIPs, ","))

		return
	}

rewrite_req:

	targetQuery := target.RawQuery

	preq.Out.URL.Scheme = target.Scheme
	preq.Out.URL.Host = target.Host
	preq.Out.URL.Path, preq.Out.URL.RawPath = joinURLPath(target, preq.Out.URL)

	if targetQuery == "" || preq.Out.URL.RawQuery == "" {
		preq.Out.URL.RawQuery = targetQuery + preq.Out.URL.RawQuery
	} else {
		preq.Out.URL.RawQuery = targetQuery + "&" + preq.Out.URL.RawQuery
	}

	// Strip X-Redfish-Url header before proxying request to target
	preq.Out.Header.Del(redfishURLHeaderName)

	// Strip Authorization header as well
	preq.Out.Header.Del(authorization)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")

	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}

	return a + b
}

func joinURLPath(a, b *url.URL) (string, string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}

	return a.Path + b.Path, apath + bpath
}
