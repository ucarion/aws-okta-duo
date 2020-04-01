package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

var errOktaNoSAMLAssertion = errors.New("okta: no SAML assertion returned from server")

type oktaAuthnRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type oktaAuthnResponse struct {
	StateToken string `json:"stateToken"`
	Embedded   struct {
		Factors []struct {
			ID string `json:"id"`
		} `json:"factors"`
	} `json:"_embedded"`
}

const oktaAuthnPath = "/api/v1/authn"

func (p Provider) oktaAuthn(ctx context.Context) (oktaAuthnResponse, error) {
	reqBody, err := json.Marshal(oktaAuthnRequest{Username: p.OktaUsername, Password: p.OktaPassword})
	if err != nil {
		return oktaAuthnResponse{}, err
	}

	reqURL := url.URL{
		Scheme: "https",
		Host:   p.OktaHost,
		Path:   oktaAuthnPath,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return oktaAuthnResponse{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return oktaAuthnResponse{}, err
	}

	defer res.Body.Close()

	var oktaAuthnResponse oktaAuthnResponse
	err = json.NewDecoder(res.Body).Decode(&oktaAuthnResponse)
	return oktaAuthnResponse, err
}

type oktaVerifyParams struct {
	FactorID   string
	StateToken string
}

type oktaVerifyRequest struct {
	StateToken string `json:"stateToken"`
}

type oktaVerifyResponse struct {
	SessionToken string `json:"sessionToken"`
	Embedded     struct {
		Factor struct {
			Embedded struct {
				Verification struct {
					Host      string `json:"host"`
					Signature string `json:"signature"`
					Links     struct {
						Complete struct {
							Href string `json:"href"`
						} `json:"complete"`
					} `json:"_links"`
				} `json:"verification"`
			} `json:"_embedded"`
		} `json:"factor"`
	} `json:"_embedded"`
}

const oktaVerifyPath = "/api/v1/authn/factors/%s/verify"

func (p Provider) oktaVerify(ctx context.Context, params oktaVerifyParams) (oktaVerifyResponse, error) {
	reqBody, err := json.Marshal(oktaVerifyRequest{StateToken: params.StateToken})
	if err != nil {
		return oktaVerifyResponse{}, err
	}

	reqURL := url.URL{
		Scheme: "https",
		Host:   p.OktaHost,
		Path:   fmt.Sprintf(oktaVerifyPath, params.FactorID),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return oktaVerifyResponse{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return oktaVerifyResponse{}, err
	}

	defer res.Body.Close()

	var oktaVerifyResponse oktaVerifyResponse
	err = json.NewDecoder(res.Body).Decode(&oktaVerifyResponse)
	return oktaVerifyResponse, err
}

type oktaCallbackParams struct {
	StateToken  string
	CallbackURL string
	Auth        string
	Signature   string
}

func (p Provider) oktaCallback(ctx context.Context, params oktaCallbackParams) error {
	reqBody := url.Values{
		"stateToken":   []string{params.StateToken},
		"sig_response": []string{fmt.Sprintf("%s:%s", params.Auth, strings.Split(params.Signature, ":")[1])},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.CallbackURL, strings.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	return nil
}

type oktaCreateSessionParams struct {
	SessionToken string
}

type oktaCreateSessionRequest struct {
	SessionToken string `json:"sessionToken"`
}

type oktaCreateSessionResponse struct {
	ID string `json:"id"`
}

const oktaSessionsPath = "/api/v1/sessions"

func (p Provider) oktaCreateSession(ctx context.Context, params oktaCreateSessionParams) (oktaCreateSessionResponse, error) {
	reqURL := url.URL{
		Scheme: "https",
		Host:   p.OktaHost,
		Path:   oktaSessionsPath,
	}

	reqBody, err := json.Marshal(oktaCreateSessionRequest{SessionToken: params.SessionToken})
	if err != nil {
		return oktaCreateSessionResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return oktaCreateSessionResponse{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return oktaCreateSessionResponse{}, err
	}

	defer res.Body.Close()

	var oktaCreateSessionResponse oktaCreateSessionResponse
	err = json.NewDecoder(res.Body).Decode(&oktaCreateSessionResponse)
	return oktaCreateSessionResponse, nil
}

type oktaGetSAMLParams struct {
	SessionID string
}

type oktaGetSAMLResponse struct {
	RawSAMLAssertion string
}

func (p Provider) oktaGetSAML(ctx context.Context, params oktaGetSAMLParams) (oktaGetSAMLResponse, error) {
	reqURL := url.URL{
		Scheme: "https",
		Host:   p.OktaHost,
		Path:   p.OktaAppPath,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return oktaGetSAMLResponse{}, err
	}

	req.AddCookie(&http.Cookie{Name: "sid", Value: params.SessionID})

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return oktaGetSAMLResponse{}, err
	}

	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	rawSAML := findNodeByName(doc, "SAMLResponse")

	if rawSAML == nil {
		return oktaGetSAMLResponse{}, errOktaNoSAMLAssertion
	}

	for _, attr := range rawSAML.Attr {
		if attr.Key == "value" {
			return oktaGetSAMLResponse{RawSAMLAssertion: attr.Val}, nil
		}
	}

	return oktaGetSAMLResponse{}, err
}
