package util

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/logger"
)

// DebugJSON helper to log JSON.
// Accepts a title to prefix the JSON log with, a *bytes.Buffer containing the JSON and a logger to use for
// logging the JSON (allowing for custom context to be added to the log).
func DebugJSON(title string, r *bytes.Buffer, l logger.Logger) {
	pretty := &bytes.Buffer{}
	err := json.Indent(pretty, r.Bytes(), "\t", "\t")
	if err != nil {
		l.Debug("Error indenting JSON", logger.Ctx{"err": err})
		return
	}

	// Print the JSON without the last "\n"
	str := pretty.String()
	l.Debug(fmt.Sprintf("%s\n\t%s", title, str[0:len(str)-1]))
}

// WriteJSON encodes the body as JSON and sends it back to the client
// Accepts optional debugLogger that activates debug logging if non-nil.
func WriteJSON(w http.ResponseWriter, body any, debugLogger logger.Logger) error {
	var output io.Writer
	var captured *bytes.Buffer

	output = w
	if debugLogger != nil {
		captured = &bytes.Buffer{}
		output = io.MultiWriter(w, captured)
	}

	enc := json.NewEncoder(output)
	enc.SetEscapeHTML(false)
	err := enc.Encode(body)

	if captured != nil {
		DebugJSON("WriteJSON", captured, debugLogger)
	}

	return err
}

// EtagHash hashes the provided data and returns the sha256.
func EtagHash(data any) (string, error) {
	etag := sha256.New()
	err := json.NewEncoder(etag).Encode(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(etag.Sum(nil)), nil
}

// EtagCheck validates the hash of the current state with the hash
// provided by the client.
func EtagCheck(r *http.Request, data any) error {
	match := r.Header.Get("If-Match")
	if match == "" {
		return nil
	}

	match = strings.Trim(match, "\"")

	hash, err := EtagHash(data)
	if err != nil {
		return err
	}

	if hash != match {
		return api.StatusErrorf(http.StatusPreconditionFailed, "ETag doesn't match: %s vs %s. The configuration has been modified since this change began. Please retrieve the updated configuration before proceeding.", hash, match)
	}

	return nil
}

// HTTPClient returns an http.Client using the given certificate and proxy.
func HTTPClient(certificate string, proxy proxyFunc) (*http.Client, error) {
	var err error
	var cert *x509.Certificate

	if certificate != "" {
		certBlock, _ := pem.Decode([]byte(certificate))
		if certBlock == nil {
			return nil, errors.New("Invalid certificate")
		}

		cert, err = x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, err
		}
	}

	tlsConfig, err := shared.GetTLSConfig(cert)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig:       tlsConfig,
		DialContext:           shared.RFC3493Dialer,
		Proxy:                 proxy,
		DisableKeepAlives:     true,
		ExpectContinueTimeout: time.Second * 30,
		ResponseHeaderTimeout: time.Second * 3600,
		TLSHandshakeTimeout:   time.Second * 5,
	}

	myhttp := http.Client{
		Transport: tr,
	}

	// Setup redirect policy
	myhttp.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Replicate the headers
		req.Header = via[len(via)-1].Header

		return nil
	}

	return &myhttp, nil
}

// A function capable of proxing an HTTP request.
type proxyFunc func(req *http.Request) (*url.URL, error)

// ContextAwareRequest is an interface implemented by http.Request starting
// from Go 1.8. It supports graceful cancellation using a context.
type ContextAwareRequest interface {
	WithContext(ctx context.Context) *http.Request
}

// certificateInDate returns an error if the current time is before the certificates "not before", or after the
// certificates "not after".
func certificateInDate(cert x509.Certificate) error {
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return api.StatusErrorf(http.StatusUnauthorized, "Certificate is not yet valid")
	}

	if now.After(cert.NotAfter) {
		return api.StatusErrorf(http.StatusUnauthorized, "Certificate has expired")
	}

	return nil
}

// CheckCASignature returns whether the certificate is signed by the CA, whether the certificate has been revoked, and the
// certificate fingerprint.
func CheckCASignature(cert x509.Certificate, networkCert *shared.CertInfo) (trusted bool, revoked bool, fingerprint string) {
	err := certificateInDate(cert)
	if err != nil {
		return false, false, ""
	}

	if networkCert == nil {
		logger.Error("Failed to check certificate has been signed by the CA, no network certificate provided")
		return false, false, ""
	}

	ca := networkCert.CA()
	if ca == nil {
		logger.Error("Failed to check certificate has been signed by the CA, no CA defined on network certificate")
		return false, false, ""
	}

	err = cert.CheckSignatureFrom(ca)
	if err != nil {
		// Certificate not signed by CA.
		return false, false, ""
	}

	crl := networkCert.CRL()
	if crl == nil {
		// No revokation list entries to check.
		return true, false, shared.CertFingerprint(&cert)
	}

	err = crl.CheckSignatureFrom(ca)
	if err != nil {
		logger.Error("Certificate revokation list has not been signed by server CA", logger.Ctx{"err": err})
		return false, false, ""
	}

	for _, revoked := range crl.RevokedCertificateEntries {
		if cert.SerialNumber.Cmp(revoked.SerialNumber) == 0 {
			// Certificate has been revoked
			return false, true, ""
		}
	}

	// Certificate not revoked.
	return true, false, shared.CertFingerprint(&cert)
}

// CheckMutualTLS checks whether the given certificate is valid and is present in the given trustedCerts map.
// Returns true if the certificate is trusted, and the fingerprint of the certificate.
func CheckMutualTLS(cert x509.Certificate, trustedCerts map[string]x509.Certificate) (bool, string) {
	err := certificateInDate(cert)
	if err != nil {
		return false, ""
	}

	// Check whether client certificate is in the map of trusted certs.
	for fingerprint, v := range trustedCerts {
		if subtle.ConstantTimeCompare(cert.Raw, v.Raw) == 1 {
			logger.Debug("Matched trusted cert", logger.Ctx{"fingerprint": fingerprint, "subject": v.Subject})
			return true, fingerprint
		}
	}

	return false, ""
}

// IsRecursionRequest checks whether the given HTTP request is marked with the
// "recursion" flag in its form values.
func IsRecursionRequest(r *http.Request) bool {
	recursionStr := r.FormValue("recursion")

	recursion, err := strconv.Atoi(recursionStr)
	if err != nil {
		return false
	}

	return recursion != 0
}

// ListenAddresses returns a list of <host>:<port> combinations at which this machine can be reached.
// It accepts the configured listen address in the following formats: <host>, <host>:<port> or :<port>.
// If a listen port is not specified then then shared.HTTPSDefaultPort is used instead.
// If a non-empty and non-wildcard host is passed in then this functions returns a single element list with the
// listen address specified. Otherwise if an empty host or wildcard address is specified then all global unicast
// addresses actively configured on the host are returned. If an IPv4 wildcard address (0.0.0.0) is specified as
// the host then only IPv4 addresses configured on the host are returned.
func ListenAddresses(configListenAddress string) ([]string, error) {
	addresses := make([]string, 0)

	if configListenAddress == "" {
		return addresses, nil
	}

	// Check if configListenAddress is a bare IP address (wrapped with square brackets or unwrapped) or a
	// hostname (without port). If so then add the default port to the configListenAddress ready for parsing.
	unwrappedConfigListenAddress := strings.Trim(configListenAddress, "[]")
	listenIP := net.ParseIP(unwrappedConfigListenAddress)
	if listenIP != nil || !strings.Contains(unwrappedConfigListenAddress, ":") {
		// Use net.JoinHostPort so that IPv6 addresses are correctly wrapped ready for parsing below.
		configListenAddress = net.JoinHostPort(unwrappedConfigListenAddress, strconv.Itoa(shared.HTTPSDefaultPort))
	}

	// By this point we should always have the configListenAddress in form <host>:<port>, so lets check that.
	// This also ensures that any wrapped IPv6 addresses are unwrapped ready for comparison below.
	localHost, localPort, err := net.SplitHostPort(configListenAddress)
	if err != nil {
		return nil, err
	}

	if localHost == "" || localHost == "0.0.0.0" || localHost == "::" {
		ifaces, err := net.Interfaces()
		if err != nil {
			return addresses, err
		}

		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				if !ip.IsGlobalUnicast() {
					continue
				}

				if ip.To4() == nil && localHost == "0.0.0.0" {
					continue
				}

				addresses = append(addresses, net.JoinHostPort(ip.String(), localPort))
			}
		}
	} else {
		addresses = append(addresses, net.JoinHostPort(localHost, localPort))
	}

	return addresses, nil
}

// GetListeners returns the socket-activated network listeners, if any.
//
// The 'start' parameter must be SystemdListenFDsStart, except in unit tests,
// see the docstring of SystemdListenFDsStart below.
func GetListeners(start int) []net.Listener {
	defer func() {
		_ = os.Unsetenv("LISTEN_PID")
		_ = os.Unsetenv("LISTEN_FDS")
	}()

	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil {
		return nil
	}

	if pid != os.Getpid() {
		return nil
	}

	fds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil {
		return nil
	}

	listeners := []net.Listener{}

	for i := start; i < start+fds; i++ {
		unix.CloseOnExec(i)

		file := os.NewFile(uintptr(i), "inherited-fd"+strconv.Itoa(i))
		listener, err := net.FileListener(file)
		if err != nil {
			continue
		}

		listeners = append(listeners, listener)
	}

	return listeners
}

// SystemdListenFDsStart is the number of the first file descriptor that might
// have been opened by systemd when socket activation is enabled. It's always 3
// in real-world usage (i.e. the first file descriptor opened after stdin,
// stdout and stderr), so this constant should always be the value passed to
// GetListeners, except for unit tests.
const SystemdListenFDsStart = 3

// IsJSONRequest returns true if the content type of the HTTP request is JSON.
func IsJSONRequest(r *http.Request) bool {
	for k, vs := range r.Header {
		if len(vs) != 1 {
			continue
		}

		if strings.ToLower(k) == "content-type" && strings.ToLower(vs[0]) == "application/json" {
			return true
		}
	}

	return false
}
