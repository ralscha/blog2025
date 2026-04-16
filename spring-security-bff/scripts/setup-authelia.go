package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const randomCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type config struct {
	clientID     string
	clientSecret string
	username     string
	password     string
	displayName  string
	email        string
	cookieDomain string
	appURL       string
	autheliaURL  string
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.clientID, "client-id", "spring-bff", "OIDC client ID to register in Authelia")
	flag.StringVar(&cfg.clientSecret, "client-secret", "spring-bff-secret", "Plaintext OIDC client secret for the Spring app")
	flag.StringVar(&cfg.username, "username", "demo", "Demo username to create in Authelia")
	flag.StringVar(&cfg.password, "password", "password", "Demo password to hash for the Authelia user")
	flag.StringVar(&cfg.displayName, "display-name", "Demo User", "Display name for the demo user")
	flag.StringVar(&cfg.email, "email", "demo@example.com", "Email address for the demo user")
	flag.StringVar(&cfg.cookieDomain, "cookie-domain", "127.0.0.1", "Authelia session cookie domain")
	flag.StringVar(&cfg.appURL, "app-url", "https://127.0.0.1:8080", "Spring Boot application base URL")
	flag.StringVar(&cfg.autheliaURL, "authelia-url", "https://127.0.0.1:9091", "Authelia issuer base URL")
	flag.Parse()
	return cfg
}

func run(cfg config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker is required to bootstrap Authelia")
	}

	rootDir, err := projectRoot()
	if err != nil {
		return err
	}

	autheliaDir := filepath.Join(rootDir, "infra", "authelia")
	dataDir := filepath.Join(autheliaDir, "data")
	secretsDir := filepath.Join(autheliaDir, "secrets")
	keyFile := filepath.Join(secretsDir, "oidc-rsa.pem")
	configurationFile := filepath.Join(autheliaDir, "configuration.yml")
	usersFile := filepath.Join(autheliaDir, "users_database.yml")

	for _, dir := range []string{autheliaDir, dataDir, secretsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	if err := ensureKeypair(secretsDir, keyFile); err != nil {
		return err
	}

	sessionSecret, err := randomSecret(64)
	if err != nil {
		return fmt.Errorf("generate session secret: %w", err)
	}
	storageKey, err := randomSecret(64)
	if err != nil {
		return fmt.Errorf("generate storage key: %w", err)
	}
	oidcHmacSecret, err := randomSecret(64)
	if err != nil {
		return fmt.Errorf("generate OIDC HMAC secret: %w", err)
	}
	resetPasswordJWTSecret, err := randomSecret(64)
	if err != nil {
		return fmt.Errorf("generate reset password JWT secret: %w", err)
	}

	clientSecretDigest, err := autheliaHash([]string{"pbkdf2", "--variant", "sha512", "--iterations", "310000"}, cfg.clientSecret)
	if err != nil {
		return fmt.Errorf("generate client secret digest: %w", err)
	}
	passwordDigest, err := autheliaHash([]string{"argon2"}, cfg.password)
	if err != nil {
		return fmt.Errorf("generate password digest: %w", err)
	}

	privateKeyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", keyFile, err)
	}

	configuration := buildConfiguration(cfg, sessionSecret, storageKey, oidcHmacSecret, resetPasswordJWTSecret, clientSecretDigest, string(privateKeyBytes))
	users := buildUsers(cfg, passwordDigest)

	if err := os.WriteFile(configurationFile, []byte(configuration), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", configurationFile, err)
	}
	if err := os.WriteFile(usersFile, []byte(users), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", usersFile, err)
	}

	fmt.Printf("Authelia configuration written to %s\n", configurationFile)
	fmt.Printf("Authelia users database written to %s\n\n", usersFile)
	fmt.Println("Spring Boot settings:")
	fmt.Printf("  AUTHELIA_ISSUER=%s\n", cfg.autheliaURL)
	fmt.Printf("  AUTHELIA_CLIENT_ID=%s\n", cfg.clientID)
	fmt.Printf("  AUTHELIA_CLIENT_SECRET=%s\n\n", cfg.clientSecret)
	fmt.Println("Demo login:")
	fmt.Printf("  username: %s\n", cfg.username)
	fmt.Printf("  password: %s\n", cfg.password)

	return nil
}

func validateConfig(cfg config) error {
	if strings.EqualFold(cfg.cookieDomain, "localhost") {
		return errors.New("cookie-domain localhost is rejected by current Authelia releases; use a dotted hostname or an IP address such as 127.0.0.1 and front the URLs with HTTPS")
	}

	if net.ParseIP(cfg.cookieDomain) == nil && !strings.Contains(cfg.cookieDomain, ".") {
		return fmt.Errorf("cookie-domain %q is not valid for Authelia; use an IP address or a hostname with at least one period", cfg.cookieDomain)
	}

	if err := validateHTTPSURL("app-url", cfg.appURL); err != nil {
		return err
	}
	if err := validateHTTPSURL("authelia-url", cfg.autheliaURL); err != nil {
		return err
	}

	return nil
}

func validateHTTPSURL(name string, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s %q is not a valid URL: %w", name, rawURL, err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("%s %q must use https for current Authelia releases", name, rawURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%s %q must include a host", name, rawURL)
	}

	return nil
}

func projectRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return workingDir, nil
}

func ensureKeypair(secretsDir string, keyFile string) error {
	if _, err := os.Stat(keyFile); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", keyFile, err)
	}

	volumePath, err := filepath.Abs(secretsDir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", secretsDir, err)
	}

	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/keys", filepath.ToSlash(volumePath)),
		"authelia/authelia:latest",
		"authelia", "crypto", "pair", "rsa", "generate", "--directory", "/keys",
	}
	if output, err := execCommand(args...); err != nil {
		return fmt.Errorf("generate RSA keypair: %w\n%s", err, output)
	}

	privateKey := filepath.Join(secretsDir, "private.pem")
	publicKey := filepath.Join(secretsDir, "public.pem")
	if _, err := os.Stat(privateKey); err != nil {
		return fmt.Errorf("authelia key generation did not create %s", privateKey)
	}
	if err := os.Rename(privateKey, keyFile); err != nil {
		return fmt.Errorf("rename %s to %s: %w", privateKey, keyFile, err)
	}
	if err := os.Remove(publicKey); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", publicKey, err)
	}

	return nil
}

func randomSecret(length int) (string, error) {
	buffer := make([]byte, length)
	for index := range buffer {
		randomIndex, err := randomInt(len(randomCharset))
		if err != nil {
			return "", err
		}
		buffer[index] = randomCharset[randomIndex]
	}
	return string(buffer), nil
}

func randomInt(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("max must be positive")
	}
	var oneByte [1]byte
	limit := 256 - (256 % max)
	for {
		if _, err := rand.Read(oneByte[:]); err != nil {
			return 0, err
		}
		value := int(oneByte[0])
		if value < limit {
			return value % max, nil
		}
	}
}

func autheliaHash(algorithmArgs []string, plainValue string) (string, error) {
	args := []string{"run", "--rm", "authelia/authelia:latest", "authelia", "crypto", "hash", "generate"}
	args = append(args, algorithmArgs...)
	args = append(args, "--password", plainValue)

	output, err := execCommand(args...)
	if err != nil {
		return "", fmt.Errorf("docker run failed: %w\n%s", err, output)
	}

	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "Digest:"); ok {
			return strings.TrimSpace(after), nil
		}
	}

	return "", fmt.Errorf("authelia did not return a digest\n%s", output)
}

func execCommand(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	return combined.String(), err
}

func buildConfiguration(cfg config, sessionSecret string, storageKey string, oidcHMACSecret string, resetPasswordJWTSecret string, clientSecretDigest string, privateKey string) string {
	indentedKey := indentMultiline(strings.TrimSpace(privateKey), "          ")
	redirectURIs := buildRedirectURIs(cfg.appURL)
	redirectURILines := make([]string, 0, len(redirectURIs))
	for _, redirectURI := range redirectURIs {
		redirectURILines = append(redirectURILines, fmt.Sprintf("          - '%s'", redirectURI))
	}

	lines := []string{
		"server:",
		"  address: tcp://0.0.0.0:9091",
		"",
		"log:",
		"  level: info",
		"",
		"theme: auto",
		"",
		"authentication_backend:",
		"  file:",
		"    path: /config/users_database.yml",
		"",
		"access_control:",
		"  default_policy: one_factor",
		"",
		"identity_validation:",
		"  reset_password:",
		fmt.Sprintf("    jwt_secret: %s", resetPasswordJWTSecret),
		"",
		"session:",
		fmt.Sprintf("  secret: %s", sessionSecret),
		"  cookies:",
		"    - name: authelia_session",
		fmt.Sprintf("      domain: %s", cfg.cookieDomain),
		fmt.Sprintf("      authelia_url: %s", cfg.autheliaURL),
		fmt.Sprintf("      default_redirection_url: %s", cfg.appURL),
		"",
		"storage:",
		fmt.Sprintf("  encryption_key: %s", storageKey),
		"  local:",
		"    path: /data/db.sqlite3",
		"",
		"notifier:",
		"  filesystem:",
		"    filename: /data/notification.txt",
		"",
		"identity_providers:",
		"  oidc:",
		fmt.Sprintf("    hmac_secret: %s", oidcHMACSecret),
		"    enforce_pkce: always",
		"    jwks:",
		"      - algorithm: RS256",
		"        use: sig",
		"        key: |",
		indentedKey,
		"    clients:",
		fmt.Sprintf("      - client_id: '%s'", cfg.clientID),
		"        client_name: 'Spring BFF Demo'",
		fmt.Sprintf("        client_secret: '%s'", clientSecretDigest),
		"        public: false",
		"        authorization_policy: one_factor",
		"        require_pkce: true",
		"        pkce_challenge_method: S256",
		"        redirect_uris:",
	}
	lines = append(lines, redirectURILines...)
	lines = append(lines,
		"        scopes:",
		"          - openid",
		"          - profile",
		"          - email",
		"          - offline_access",
		"        grant_types:",
		"          - authorization_code",
		"        response_types:",
		"          - code",
		"        token_endpoint_auth_method: client_secret_basic",
	)

	return strings.Join(lines, "\n")
}

func buildRedirectURIs(appURL string) []string {
	parsed, err := url.Parse(appURL)
	if err != nil {
		return []string{strings.TrimRight(appURL, "/") + "/login/oauth2/code/authelia"}
	}

	callbackURL := *parsed
	callbackURL.RawQuery = ""
	callbackURL.Fragment = ""
	callbackURL.Path = strings.TrimRight(parsed.Path, "/") + "/login/oauth2/code/authelia"

	redirectURIs := []string{callbackURL.String()}
	if alternateHost := alternateLoopbackHost(parsed.Hostname()); alternateHost != "" {
		alternateURL := callbackURL
		if port := parsed.Port(); port != "" {
			alternateURL.Host = net.JoinHostPort(alternateHost, port)
		} else {
			alternateURL.Host = alternateHost
		}
		redirectURIs = append(redirectURIs, alternateURL.String())
	}

	return uniqueStrings(redirectURIs)
}

func alternateLoopbackHost(host string) string {
	switch host {
	case "127.0.0.1":
		return "localhost"
	case "localhost":
		return "127.0.0.1"
	default:
		return ""
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func buildUsers(cfg config, passwordDigest string) string {
	lines := []string{
		"# yaml-language-server: $schema=https://www.authelia.com/schemas/latest/json-schema/user-database.json",
		"users:",
		fmt.Sprintf("  %s:", cfg.username),
		"    disabled: false",
		fmt.Sprintf("    displayname: '%s'", escapeSingleQuotes(cfg.displayName)),
		fmt.Sprintf("    password: '%s'", passwordDigest),
		fmt.Sprintf("    email: '%s'", escapeSingleQuotes(cfg.email)),
		"    groups:",
		"      - users",
	}

	return strings.Join(lines, "\n")
}

func indentMultiline(value string, indent string) string {
	parts := strings.Split(value, "\n")
	for index, part := range parts {
		parts[index] = indent + strings.TrimRight(part, "\r")
	}
	return strings.Join(parts, "\n")
}

func escapeSingleQuotes(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
