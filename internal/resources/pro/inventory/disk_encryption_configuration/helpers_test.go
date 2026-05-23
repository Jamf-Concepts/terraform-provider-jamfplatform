// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

// Test fixtures for the disk_encryption_configuration acceptance suite.
// Generates a self-signed DER public-cert blob in memory so the
// acceptance tests can exercise the IRK upload path without checking
// sample certs into the repo.
//
// PKCS12 fixture (password-bearing path) is intentionally NOT generated
// here — that would require an extra dependency (e.g.
// software.sslmate.com/src/go-pkcs12) and the project policy is to
// avoid go.mod churn for test-only deps. The password write-only
// behaviour is exercised by the unit tests in
// `state_builders_test.go` and `input_builders_test.go` (which use
// hand-built struct fixtures). Live PKCS12 round-tripping is covered
// by manual verification — see local-testing/diskencryption/.
//
// Build tag is `acceptance` so the generator only compiles into the
// acceptance binary — keeps `make test` lean.

package disk_encryption_configuration_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"sync"
	"testing"
	"time"
)

// fixtureOnce caches the generated cert so multiple tests share the
// same fixture without paying the keygen cost per test.
var fixtureOnce sync.Once
var (
	cachedDERB64 string
	cachedErr    error
)

// loadDERFixture returns the base64-encoded DER public-cert blob used
// by the acceptance tests. Generated on first call, cached for the
// rest of the run.
func loadDERFixture(t *testing.T) string {
	t.Helper()
	fixtureOnce.Do(generateFixture)
	if cachedErr != nil {
		t.Fatalf("disk_encryption_configuration acceptance fixture: %v", cachedErr)
	}
	return cachedDERB64
}

func generateFixture() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		cachedErr = err
		return
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "tf-acc-disk-encryption", Organization: []string{"jamf-tf-provider"}, Country: []string{"US"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		cachedErr = err
		return
	}
	cachedDERB64 = base64.StdEncoding.EncodeToString(derBytes)
}
