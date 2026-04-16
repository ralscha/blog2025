package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	truststorePassword = "changeit"
	truststoreAlias    = "bff-local-dev-ca"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}

	httpsDir := filepath.Join(rootDir, "infra", "https")
	if err := os.MkdirAll(httpsDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", httpsDir, err)
	}

	caPath := filepath.Join(httpsDir, "ca.pem")
	caKeyPath := filepath.Join(httpsDir, "ca-key.pem")
	tlsPath := filepath.Join(httpsDir, "tls.pem")
	tlsKeyPath := filepath.Join(httpsDir, "tls-key.pem")
	truststorePath := filepath.Join(httpsDir, "truststore.p12")

	if existingAssetsPresent(caPath, caKeyPath, tlsPath, tlsKeyPath) {
		if err := ensureTruststore(caPath, truststorePath); err != nil {
			return err
		}
		printSettings(httpsDir)
		return nil
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("generate CA private key: %w", err)
	}

	caTemplate, err := certificateTemplate("bff-local-dev-ca")
	if err != nil {
		return err
	}
	caTemplate.IsCA = true
	caTemplate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	caTemplate.BasicConstraintsValid = true
	caTemplate.Subject = pkix.Name{CommonName: "bff-local-dev-ca"}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA certificate: %w", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate server private key: %w", err)
	}

	serverTemplate, err := certificateTemplate("127.0.0.1")
	if err != nil {
		return err
	}
	serverTemplate.Subject = pkix.Name{CommonName: "127.0.0.1"}
	serverTemplate.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	serverTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	serverTemplate.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	serverTemplate.DNSNames = []string{"localhost"}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create server certificate: %w", err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		return fmt.Errorf("write ca.pem: %w", err)
	}
	if err := os.WriteFile(caKeyPath, caKeyPEM, 0o600); err != nil {
		return fmt.Errorf("write ca-key.pem: %w", err)
	}
	if err := os.WriteFile(tlsPath, append(serverPEM, caPEM...), 0o600); err != nil {
		return fmt.Errorf("write tls.pem: %w", err)
	}
	if err := os.WriteFile(tlsKeyPath, serverKeyPEM, 0o600); err != nil {
		return fmt.Errorf("write tls-key.pem: %w", err)
	}

	if err := ensureTruststore(caPath, truststorePath); err != nil {
		return err
	}

	printSettings(httpsDir)

	return nil
}

func existingAssetsPresent(paths ...string) bool {
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func ensureTruststore(caPath string, truststorePath string) error {
	if _, err := exec.LookPath("keytool"); err != nil {
		return fmt.Errorf("keytool is required to build the Java truststore: %w", err)
	}

	if err := os.Remove(truststorePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reset truststore: %w", err)
	}

	cmd := exec.Command("keytool",
		"-importcert",
		"-noprompt",
		"-alias", truststoreAlias,
		"-file", caPath,
		"-keystore", truststorePath,
		"-storepass", truststorePassword,
		"-storetype", "PKCS12",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create truststore: %w", err)
	}

	return nil
}

func printSettings(httpsDir string) {
	fmt.Printf("Local HTTPS assets written to %s\n", httpsDir)
	fmt.Println("Spring Boot local HTTPS settings:")
	fmt.Println("  SERVER_PORT=8081")
	fmt.Println("  AUTHELIA_ISSUER=https://127.0.0.1:9091")
	fmt.Printf("  JAVA_TOOL_OPTIONS=-Djavax.net.ssl.trustStore=%s -Djavax.net.ssl.trustStorePassword=%s -Djavax.net.ssl.trustStoreType=PKCS12\n", filepath.ToSlash(filepath.Join("infra", "https", "truststore.p12")), truststorePassword)
	fmt.Println("Windows trust command:")
	fmt.Println("  Import-Certificate -FilePath .\\infra\\https\\ca.pem -CertStoreLocation Cert:\\CurrentUser\\Root")
}

func certificateTemplate(commonName string) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}

	now := time.Now().UTC()
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(5, 0, 0),
		BasicConstraintsValid: true,
	}, nil
}
