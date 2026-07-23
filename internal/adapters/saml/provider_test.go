package saml

import (
	"strings"
	"testing"

	"github.com/crewjam/saml"
	"github.com/curefatih/afi/internal/identity"
)

const testIDPMetadata = `<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="http://idp.example.com/metadata">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIICajCCAdOgAwIBAgIBADANBgkqhkiG9w0BAQ0FADBSMQswCQYDVQQGEwJ1czETMBEGA1UECAwKQ2FsaWZvcm5pYTEVMBMGA1UECgwMT25lbG9naW4gSW5jMRcwFQYDVQQDDA5zcC5leGFtcGxlLmNvbTAeFw0xNDA3MTcxNDEyNTZaFw0xNTA3MTcxNDEyNTZaMFIxCzAJBgNVBAYTAnVzMRMwEQYDVQQIDApDYWxpZm9ybmlhMRUwEwYDVQQKDAxPbmVsb2dpbiBJbmMxFzAVBgNVBAMMDnNwLmV4YW1wbGUuY29tMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDZx+ON4IUoIWxgukTb1tOiX3bMYzYQiwWPUNMp+Fq82xoNogso2bykZG0yiJm5o8zv/sd6pGouayMgkx/2FSOdc36T0jGbCHuRSbtia0PEzNIRtmViMrt3AeoWBidRXmZsxCNLwgIV6dn2WpuE5Az0bHgpZnQxTKFek0BMKU/d8wIDAQABo1AwTjAdBgNVHQ4EFgQUGHxYqZYyX7cTxKVODVgZwSTdCnwwHwYDVR0jBBgwFoAUGHxYqZYyX7cTxKVODVgZwSTdCnwwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQ0FAAOBgQByFOl+hMFICbd3DJfnp2Rgd/dqttsZG/tyhILWvErbio/DEe98mXpowhTkC04ENprOyXi7ZbUqiicF89uAGyt1oqgTUCD1VsLahqIcmrzgumNyTwLGWo17WDAa1/usDhetWAMhgzF/Cnf5ek0nK00m0YZGyc4LzgD0CROMASTWNg==</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`

func TestNewRequiresMetadata(t *testing.T) {
	t.Parallel()
	_, err := New(Config{
		ID: "okta", ACSURL: "http://cp/callback", MetadataURL: "http://cp/metadata",
	})
	if err == nil {
		t.Fatal("expected metadata error")
	}
}

func TestNewAndAuthURLAndMetadata(t *testing.T) {
	t.Parallel()
	p, err := New(Config{
		ID:             "okta",
		DisplayName:    "Okta",
		ACSURL:         "http://localhost:8081/api/v1/platform/auth/sso/okta/callback",
		MetadataURL:    "http://localhost:8081/api/v1/platform/auth/sso/okta/metadata",
		IDPMetadataXML: testIDPMetadata,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !p.EphemeralKey() {
		t.Fatal("expected ephemeral key when PEMs omitted")
	}
	if p.Type() != "saml" || p.Name() != "okta" {
		t.Fatalf("got type=%s name=%s", p.Type(), p.Name())
	}
	authURL, reqID, err := p.AuthURLWithID("relay-state-1", p.cfg.ACSURL)
	if err != nil {
		t.Fatal(err)
	}
	if reqID == "" || !strings.Contains(authURL, "SAMLRequest=") || !strings.Contains(authURL, "RelayState=relay-state-1") {
		t.Fatalf("authURL=%s reqID=%s", authURL, reqID)
	}
	raw, err := p.MetadataXML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "EntityDescriptor") || !strings.Contains(string(raw), "SPSSODescriptor") {
		t.Fatalf("metadata=%s", raw)
	}
}

func TestClaimsFromAssertionEmailNameID(t *testing.T) {
	t.Parallel()
	cfg := Config{ID: "okta", RequireEmailVerified: true}
	assertion := &saml.Assertion{
		Issuer: saml.Issuer{Value: "http://idp.example.com/metadata"},
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Format: string(saml.EmailAddressNameIDFormat),
				Value:  "user@example.com",
			},
		},
		AttributeStatements: []saml.AttributeStatement{{
			Attributes: []saml.Attribute{{
				Name:   "displayName",
				Values: []saml.AttributeValue{{Value: "Ada"}},
			}},
		}},
	}
	claims, err := claimsFromAssertion(cfg, assertion)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user@example.com" || claims.Email != "user@example.com" || !claims.EmailVerified || claims.Name != "Ada" {
		t.Fatalf("%+v", claims)
	}
}

func TestClaimsFromAssertionEmailAttribute(t *testing.T) {
	t.Parallel()
	cfg := Config{ID: "okta", RequireEmailVerified: true}
	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{Value: "opaque-sub"},
		},
		AttributeStatements: []saml.AttributeStatement{{
			Attributes: []saml.Attribute{{
				Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
				Values: []saml.AttributeValue{{Value: "attr@example.com"}},
			}},
		}},
	}
	claims, err := claimsFromAssertion(cfg, assertion)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "opaque-sub" || claims.Email != "attr@example.com" {
		t.Fatalf("%+v", claims)
	}
}

func TestClaimsRequireEmail(t *testing.T) {
	t.Parallel()
	cfg := Config{ID: "okta", RequireEmailVerified: true}
	_, err := claimsFromAssertion(cfg, &saml.Assertion{
		Subject: &saml.Subject{NameID: &saml.NameID{Value: "opaque"}},
	})
	if err != identity.ErrEmailNotVerified {
		t.Fatalf("got %v", err)
	}
}
