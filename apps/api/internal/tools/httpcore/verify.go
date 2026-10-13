package httpcore

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

// RootsForTests, when set, replaces the system trust store for verification reports.
var RootsForTests *x509.CertPool

// VerifyChain verifies the peer chain for name (the client always connects with
// InsecureSkipVerify so broken certificates can be inspected; this reports the verdict).
func VerifyChain(st *tls.ConnectionState, name string) error {
	if len(st.PeerCertificates) == 0 {
		return errors.New("no certificate presented")
	}
	inter := x509.NewCertPool()
	for _, c := range st.PeerCertificates[1:] {
		inter.AddCert(c)
	}
	_, err := st.PeerCertificates[0].Verify(x509.VerifyOptions{DNSName: name, Intermediates: inter, Roots: RootsForTests})
	if err != nil {
		var hn x509.HostnameError
		var ua x509.UnknownAuthorityError
		var inv x509.CertificateInvalidError
		switch {
		case errors.As(err, &hn):
			return fmt.Errorf("certificate is not valid for %s", name)
		case errors.As(err, &ua):
			return errors.New("certificate is not signed by a trusted authority")
		case errors.As(err, &inv):
			return fmt.Errorf("certificate is invalid: %s", inv.Error())
		}
		return fmt.Errorf("verification failed: %w", err)
	}
	return nil
}
