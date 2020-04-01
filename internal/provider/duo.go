package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type duoAuthParams struct {
	Signature string
	Host      string
}

type duoAuthResult struct {
	SessionID string
}

const duoAuthPath = "/frame/web/v1/auth"
const duoDummyParent = "http://0.0.0.0:3000/duo"
const duoDummyVersion = "2.1"

func (p Provider) duoAuth(ctx context.Context, params duoAuthParams) (duoAuthResult, error) {
	tx := strings.Split(params.Signature, ":")[0]
	reqURL := url.URL{
		Scheme: "https",
		Host:   params.Host,
		Path:   duoAuthPath,
		RawQuery: url.Values{
			"tx":     []string{tx},
			"parent": []string{duoDummyParent},
			"v":      []string{duoDummyVersion},
		}.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), nil)
	if err != nil {
		return duoAuthResult{}, err
	}

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return duoAuthResult{}, err
	}

	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	if err != nil {
		return duoAuthResult{}, err
	}

	sid := findNodeByName(doc, "sid")
	for _, attr := range sid.Attr {
		if attr.Key == "value" {
			return duoAuthResult{SessionID: attr.Val}, nil
		}
	}

	return duoAuthResult{}, nil
}

type duoPromptParams struct {
	Host      string
	SessionID string
}

type duoPromptResponse struct {
	Response struct {
		TxID string `json:"txid"`
	} `json:"response"`
}

func (p Provider) duoPrompt(ctx context.Context, params duoPromptParams) (duoPromptResponse, error) {
	reqBody := url.Values{
		"sid":         []string{params.SessionID},
		"device":      []string{p.DuoDevice},
		"factor":      []string{"Duo Push"},
		"out_of_date": []string{"False"},
	}.Encode()

	reqURL := url.URL{
		Scheme: "https",
		Host:   params.Host,
		Path:   "/frame/prompt",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), strings.NewReader(reqBody))
	if err != nil {
		return duoPromptResponse{}, err
	}

	req.Header.Add("Origin", fmt.Sprintf("https://%s", params.Host))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return duoPromptResponse{}, err
	}

	defer res.Body.Close()

	var duoPromptResponse duoPromptResponse
	err = json.NewDecoder(res.Body).Decode(&duoPromptResponse)
	return duoPromptResponse, nil
}

type duoStatusParams struct {
	Host      string
	SessionID string
	TxID      string
}

type duoStatusResponse struct {
	Response struct {
		Result    string `json:"result"`
		ResultURL string `json:"result_url"`
		Cookie    string `json:"cookie"`
	} `json:"response"`
}

func (p Provider) duoStatus(ctx context.Context, params duoStatusParams) (duoStatusResponse, error) {
	reqBody := url.Values{
		"sid":  []string{params.SessionID},
		"txid": []string{params.TxID},
	}.Encode()

	reqURL := url.URL{
		Scheme: "https",
		Host:   params.Host,
		Path:   "/frame/status",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), strings.NewReader(reqBody))
	if err != nil {
		return duoStatusResponse{}, err
	}

	req.Header.Add("Origin", fmt.Sprintf("https://%s", params.Host))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return duoStatusResponse{}, err
	}

	defer res.Body.Close()

	var duoStatusResponse duoStatusResponse
	err = json.NewDecoder(res.Body).Decode(&duoStatusResponse)
	return duoStatusResponse, nil
}

type duoStatusRedirectParams struct {
	Host      string
	SessionID string
	Path      string
}

func (p Provider) duoStatusRedirect(ctx context.Context, params duoStatusRedirectParams) (duoStatusResponse, error) {
	reqBody := url.Values{"sid": []string{params.SessionID}}.Encode()
	reqURL := url.URL{
		Scheme: "https",
		Host:   params.Host,
		Path:   params.Path,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), strings.NewReader(reqBody))
	if err != nil {
		return duoStatusResponse{}, err
	}

	req.Header.Add("Origin", fmt.Sprintf("https://%s", params.Host))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	res, err := p.HTTPClient.Do(req)
	if err != nil {
		return duoStatusResponse{}, err
	}

	defer res.Body.Close()

	var duoStatusResponse duoStatusResponse
	err = json.NewDecoder(res.Body).Decode(&duoStatusResponse)
	return duoStatusResponse, nil
}
