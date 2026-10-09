package webeditor

import (
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// remotePath is the only route that clients on other hosts may use: the
// Claude Code hook posts status from WSL, which reaches the editor through the
// host's network address rather than loopback. This is a deliberate product
// decision: the route is unauthenticated, so anything that can reach the port
// can read and set the Claude status (see profiles/CLAUDE_CODE_HOOKS.md).
const remotePath = "/api/claude-status"

// withAccessPolicy wraps the editor's routes with one access policy:
//   - every route except remotePath accepts only loopback clients that address
//     the server by a loopback name (the Host check defeats DNS rebinding);
//   - requests that change state must not come from a foreign web page: an
//     Origin header, when present, must be a loopback http origin on the
//     server's port.
func withAccessPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != remotePath && (!isLoopbackAddr(r.RemoteAddr) || !isLoopbackHostname(hostname(r.Host))) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if !isSafeMethod(r.Method) && !isTrustedOrigin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isSafeMethod reports whether the method only reads.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// isTrustedOrigin reports whether the request's Origin, if any, is a page
// served by this editor: http, a loopback host and the same port as the
// request's Host. Requests without Origin are not sent cross-site by browsers.
func isTrustedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.Path != "" || u.User != nil {
		return false
	}
	return isLoopbackHostname(u.Hostname()) && u.Port() == port(r.Host)
}

// isLoopbackAddr reports whether a "host:port" remote address is a loopback IP.
func isLoopbackAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isLoopbackHostname reports whether a host name refers to this machine's
// loopback interface.
func isLoopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// hostname returns the host part of a Host header value ("[::1]:8384" -> "::1").
func hostname(hostport string) string {
	return (&url.URL{Host: hostport}).Hostname()
}

// port returns the port part of a Host header value, or "" if there is none.
func port(hostport string) string {
	return (&url.URL{Host: hostport}).Port()
}

// isKnownProfile reports whether path is one of the profiles the profile
// provider lists. Paths sent by the browser are only used when they match,
// so the editor cannot be used to read or write other files.
func (s *Server) isKnownProfile(path string) bool {
	if s.profileProvider == nil || path == "" {
		return false
	}
	clean := filepath.Clean(path)
	for _, p := range s.profileProvider.GetProfiles() {
		if filepath.Clean(p.Path) == clean {
			return true
		}
	}
	return false
}
