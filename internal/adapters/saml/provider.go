package saml

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/curefatih/afi/internal/identity"
)

// Config describes a platform-wide SAML 2.0 IdP / SP pair.
type Config struct {
	ID                   string
	DisplayName          string
	EntityID             string // SP entity ID; defaults to MetadataURL
	MetadataURL          string // public SP metadata URL
	ACSURL               string // Assertion Consumer Service (callback) URL
	IDPMetadataURL       string
	IDPMetadataXML       string
	SPCertPEM            string
	SPKeyPEM             string
	RequireEmailVerified bool
	AllowIDPInitiated    bool
	HTTPClient           *http.Client
	// EphemeralKey is set when SP key material was generated at startup (dev only).
	EphemeralKey bool
}

// Provider implements identity.FederationProvider for SAML 2.0.
type Provider struct {
	cfg Config
	sp  *saml.ServiceProvider
}

// New builds a SAML SP from IdP metadata and optional SP key material.
func New(cfg Config) (*Provider, error) {
	cfg.ID = strings.TrimSpace(cfg.ID)
	if cfg.ID == "" {
		return nil, fmt.Errorf("sso provider: id is required")
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = cfg.ID
	}
	cfg.ACSURL = strings.TrimSpace(cfg.ACSURL)
	cfg.MetadataURL = strings.TrimSpace(cfg.MetadataURL)
	if cfg.ACSURL == "" || cfg.MetadataURL == "" {
		return nil, fmt.Errorf("sso provider %q: acs_url and metadata_url are required", cfg.ID)
	}
	cfg.IDPMetadataURL = strings.TrimSpace(cfg.IDPMetadataURL)
	cfg.IDPMetadataXML = strings.TrimSpace(cfg.IDPMetadataXML)
	if cfg.IDPMetadataURL == "" && cfg.IDPMetadataXML == "" {
		return nil, fmt.Errorf("sso provider %q: idp_metadata_url or idp_metadata_xml is required", cfg.ID)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	key, cert, ephemeral, err := loadOrGenerateKey(cfg.SPCertPEM, cfg.SPKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("sso provider %q: sp key: %w", cfg.ID, err)
	}
	cfg.EphemeralKey = ephemeral
	cfg.SPCertPEM = ""
	cfg.SPKeyPEM = ""

	idpMeta, err := loadIDPMetadata(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("sso provider %q: idp metadata: %w", cfg.ID, err)
	}

	acsURL, err := url.Parse(cfg.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("sso provider %q: acs_url: %w", cfg.ID, err)
	}
	metaURL, err := url.Parse(cfg.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("sso provider %q: metadata_url: %w", cfg.ID, err)
	}
	entityID := strings.TrimSpace(cfg.EntityID)
	if entityID == "" {
		entityID = metaURL.String()
	}

	sp := &saml.ServiceProvider{
		EntityID:          entityID,
		Key:               key,
		Certificate:       cert,
		MetadataURL:       *metaURL,
		AcsURL:            *acsURL,
		IDPMetadata:       idpMeta,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
		AllowIDPInitiated: cfg.AllowIDPInitiated,
	}

	return &Provider{cfg: cfg, sp: sp}, nil
}

func (p *Provider) Name() string        { return p.cfg.ID }
func (p *Provider) DisplayName() string { return p.cfg.DisplayName }
func (p *Provider) Type() string        { return "saml" }

// EphemeralKey reports whether the SP signing key was generated at process start.
func (p *Provider) EphemeralKey() bool { return p.cfg.EphemeralKey }

// AuthURL builds an HTTP-Redirect AuthnRequest URL (RelayState = state).
func (p *Provider) AuthURL(state, redirectURI string) (string, error) {
	u, _, err := p.AuthURLWithID(state, redirectURI)
	return u, err
}

// AuthURLWithID returns the IdP redirect URL and AuthnRequest ID.
func (p *Provider) AuthURLWithID(state, redirectURI string) (authURL, requestID string, err error) {
	if p == nil || p.sp == nil {
		return "", "", fmt.Errorf("saml provider not configured")
	}
	_ = redirectURI // ACS is fixed on the SP; BeginSSO still passes the public callback URL for parity.
	bindingLoc := p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	binding := saml.HTTPRedirectBinding
	if bindingLoc == "" {
		bindingLoc = p.sp.GetSSOBindingLocation(saml.HTTPPostBinding)
		binding = saml.HTTPPostBinding
	}
	if bindingLoc == "" {
		return "", "", fmt.Errorf("sso provider %q: idp metadata has no SSO binding", p.cfg.ID)
	}
	if binding == saml.HTTPPostBinding {
		// AuthURL must be a browser redirect; require HTTP-Redirect binding.
		return "", "", fmt.Errorf("sso provider %q: idp metadata requires HTTP-Redirect SSO binding", p.cfg.ID)
	}
	req, err := p.sp.MakeAuthenticationRequest(bindingLoc, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", err
	}
	u, err := req.Redirect(state, p.sp)
	if err != nil {
		return "", "", err
	}
	return u.String(), req.ID, nil
}

// Exchange maps SAMLResponse (base64) into claims. Prefer ExchangeAssertion from ACS.
func (p *Provider) Exchange(ctx context.Context, code, redirectURI string) (identity.FederatedClaims, error) {
	ids := []string{""}
	return p.ExchangeAssertion(ctx, code, redirectURI, ids)
}

// ExchangeAssertion validates a SAMLResponse and maps it to FederatedClaims.
func (p *Provider) ExchangeAssertion(ctx context.Context, response, redirectURI string, possibleRequestIDs []string) (identity.FederatedClaims, error) {
	_ = ctx
	if p == nil || p.sp == nil {
		return identity.FederatedClaims{}, fmt.Errorf("saml provider not configured")
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return identity.FederatedClaims{}, fmt.Errorf("empty SAMLResponse")
	}
	form := url.Values{}
	form.Set("SAMLResponse", response)
	req, err := http.NewRequest(http.MethodPost, p.cfg.ACSURL, strings.NewReader(form.Encode()))
	if err != nil {
		return identity.FederatedClaims{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		return identity.FederatedClaims{}, err
	}
	if len(possibleRequestIDs) == 0 {
		possibleRequestIDs = []string{""}
	}
	assertion, err := p.sp.ParseResponse(req, possibleRequestIDs)
	if err != nil {
		return identity.FederatedClaims{}, fmt.Errorf("validate saml response: %w", err)
	}
	_ = redirectURI
	return claimsFromAssertion(p.cfg, assertion)
}

// MetadataXML returns the SP EntityDescriptor.
func (p *Provider) MetadataXML() ([]byte, error) {
	if p == nil || p.sp == nil {
		return nil, fmt.Errorf("saml provider not configured")
	}
	return xml.MarshalIndent(p.sp.Metadata(), "", "  ")
}

func claimsFromAssertion(cfg Config, assertion *saml.Assertion) (identity.FederatedClaims, error) {
	if assertion == nil {
		return identity.FederatedClaims{}, fmt.Errorf("empty assertion")
	}
	subject := ""
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		subject = strings.TrimSpace(assertion.Subject.NameID.Value)
	}
	email := emailFromAssertion(assertion)
	name := attributeValue(assertion, "displayName", "name", "cn",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		"http://schemas.microsoft.com/identity/claims/displayname",
	)
	issuer := strings.TrimSpace(assertion.Issuer.Value)
	if subject == "" {
		return identity.FederatedClaims{}, identity.ErrMissingFederatedSubject
	}
	verified := email != ""
	if cfg.RequireEmailVerified && !verified {
		return identity.FederatedClaims{}, identity.ErrEmailNotVerified
	}
	return identity.FederatedClaims{
		Provider:      cfg.ID,
		Issuer:        issuer,
		Subject:       subject,
		Email:         email,
		EmailVerified: verified,
		Name:          name,
	}, nil
}

func emailFromAssertion(assertion *saml.Assertion) string {
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		nid := assertion.Subject.NameID
		format := strings.TrimSpace(nid.Format)
		val := strings.TrimSpace(nid.Value)
		if val != "" && (format == "" ||
			strings.EqualFold(format, string(saml.EmailAddressNameIDFormat)) ||
			strings.Contains(strings.ToLower(format), "email")) {
			if strings.Contains(val, "@") {
				return val
			}
		}
	}
	return attributeValue(assertion,
		"email", "mail", "Email",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/email",
		"urn:oid:0.9.2342.19200300.100.1.3",
	)
}

func attributeValue(assertion *saml.Assertion, names ...string) string {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[strings.ToLower(n)] = struct{}{}
	}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			name := strings.ToLower(strings.TrimSpace(attr.Name))
			friendly := strings.ToLower(strings.TrimSpace(attr.FriendlyName))
			if _, ok := want[name]; !ok {
				if _, ok := want[friendly]; !ok {
					continue
				}
			}
			for _, v := range attr.Values {
				if s := strings.TrimSpace(v.Value); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func loadIDPMetadata(ctx context.Context, cfg Config) (*saml.EntityDescriptor, error) {
	if cfg.IDPMetadataXML != "" {
		return samlsp.ParseMetadata([]byte(cfg.IDPMetadataXML))
	}
	u, err := url.Parse(cfg.IDPMetadataURL)
	if err != nil {
		return nil, err
	}
	return samlsp.FetchMetadata(ctx, cfg.HTTPClient, *u)
}

func loadOrGenerateKey(certPEM, keyPEM string) (key *rsa.PrivateKey, cert *x509.Certificate, ephemeral bool, err error) {
	certPEM = strings.TrimSpace(certPEM)
	keyPEM = strings.TrimSpace(keyPEM)
	if certPEM != "" || keyPEM != "" {
		if certPEM == "" || keyPEM == "" {
			return nil, nil, false, fmt.Errorf("both sp_cert_pem and sp_key_pem are required when either is set")
		}
		keyBlock, _ := pem.Decode([]byte(keyPEM))
		if keyBlock == nil {
			return nil, nil, false, fmt.Errorf("invalid sp_key_pem")
		}
		parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			parsedKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			if err != nil {
				return nil, nil, false, fmt.Errorf("parse sp_key_pem: %w", err)
			}
		}
		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, false, fmt.Errorf("sp_key_pem must be RSA")
		}
		certBlock, _ := pem.Decode([]byte(certPEM))
		if certBlock == nil {
			return nil, nil, false, fmt.Errorf("invalid sp_cert_pem")
		}
		parsedCert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, nil, false, fmt.Errorf("parse sp_cert_pem: %w", err)
		}
		return rsaKey, parsedCert, false, nil
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, false, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "afi-saml-sp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		return nil, nil, false, err
	}
	parsedCert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, false, err
	}
	return rsaKey, parsedCert, true, nil
}

var (
	_ identity.FederationProvider  = (*Provider)(nil)
	_ identity.AuthStarter         = (*Provider)(nil)
	_ identity.AssertionExchanger  = (*Provider)(nil)
	_ identity.ServiceProviderMeta = (*Provider)(nil)
)
