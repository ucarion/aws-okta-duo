package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go/service/sts"
)

type Provider struct {
	OktaSessionID string
	OktaHost      string
	OktaUsername  string
	OktaPassword  string
	OktaAppPath   string
	DuoDevice     string
	STSClient     *sts.STS
	HTTPClient    *http.Client
}

type GetCredentialsResult struct {
	Credentials   *sts.Credentials
	OktaSessionID string
}

var errDuoStatusFailed = errors.New("duo: login request failed")

func (p Provider) GetCredentials(ctx context.Context) (GetCredentialsResult, error) {
	if p.OktaSessionID != "" {
		creds, err := p.getCredentialsFromOktaSessionID(ctx, p.OktaSessionID)
		if err == nil {
			return GetCredentialsResult{OktaSessionID: p.OktaSessionID, Credentials: creds}, nil
		}
	}

	oktaSessionID, err := p.getOktaSessionID(ctx)
	if err != nil {
		return GetCredentialsResult{}, err
	}

	creds, err := p.getCredentialsFromOktaSessionID(ctx, oktaSessionID)
	if err != nil {
		return GetCredentialsResult{}, err
	}

	return GetCredentialsResult{OktaSessionID: oktaSessionID, Credentials: creds}, nil
}

func (p Provider) getOktaSessionID(ctx context.Context) (string, error) {
	oktaAuthnRes, err := p.oktaAuthn(ctx)
	if err != nil {
		return "", err
	}

	oktaVerifyRes, err := p.oktaVerify(ctx, oktaVerifyParams{
		StateToken: oktaAuthnRes.StateToken,
		FactorID:   oktaAuthnRes.Embedded.Factors[0].ID,
	})

	if err != nil {
		return "", err
	}

	duoAuthRes, err := p.duoAuth(ctx, duoAuthParams{
		Host:      oktaVerifyRes.Embedded.Factor.Embedded.Verification.Host,
		Signature: oktaVerifyRes.Embedded.Factor.Embedded.Verification.Signature,
	})

	if err != nil {
		return "", err
	}

	duoPromptRes, err := p.duoPrompt(ctx, duoPromptParams{
		Host:      oktaVerifyRes.Embedded.Factor.Embedded.Verification.Host,
		SessionID: duoAuthRes.SessionID,
	})

	if err != nil {
		return "", err
	}

	// Initial request will always immediately return nothing of interest.
	_, err = p.duoStatus(ctx, duoStatusParams{
		Host:      oktaVerifyRes.Embedded.Factor.Embedded.Verification.Host,
		SessionID: duoAuthRes.SessionID,
		TxID:      duoPromptRes.Response.TxID,
	})

	if err != nil {
		return "", err
	}

	// Second request will block until the push is accepted / rejected.
	duoStatusRes, err := p.duoStatus(ctx, duoStatusParams{
		Host:      oktaVerifyRes.Embedded.Factor.Embedded.Verification.Host,
		SessionID: duoAuthRes.SessionID,
		TxID:      duoPromptRes.Response.TxID,
	})

	if err != nil {
		return "", err
	}

	if duoStatusRes.Response.Result != "SUCCESS" {
		return "", errDuoStatusFailed
	}

	duoStatusRedirectRes, err := p.duoStatusRedirect(ctx, duoStatusRedirectParams{
		Host:      oktaVerifyRes.Embedded.Factor.Embedded.Verification.Host,
		SessionID: duoAuthRes.SessionID,
		Path:      duoStatusRes.Response.ResultURL,
	})

	if err != nil {
		return "", err
	}

	err = p.oktaCallback(ctx, oktaCallbackParams{
		StateToken:  oktaAuthnRes.StateToken,
		CallbackURL: oktaVerifyRes.Embedded.Factor.Embedded.Verification.Links.Complete.Href,
		Signature:   oktaVerifyRes.Embedded.Factor.Embedded.Verification.Signature,
		Auth:        duoStatusRedirectRes.Response.Cookie,
	})

	if err != nil {
		return "", err
	}

	oktaVerifyRes, err = p.oktaVerify(ctx, oktaVerifyParams{
		StateToken: oktaAuthnRes.StateToken,
		FactorID:   oktaAuthnRes.Embedded.Factors[0].ID,
	})

	if err != nil {
		return "", err
	}

	oktaSessionRes, err := p.oktaCreateSession(ctx, oktaCreateSessionParams{
		SessionToken: oktaVerifyRes.SessionToken,
	})

	if err != nil {
		return "", err

	}

	return oktaSessionRes.ID, nil
}

func (p Provider) getCredentialsFromOktaSessionID(ctx context.Context, oktaSessionID string) (*sts.Credentials, error) {
	oktaSAMLRes, err := p.oktaGetSAML(ctx, oktaGetSAMLParams{SessionID: oktaSessionID})
	if err != nil {
		return nil, err
	}

	decodedSAMLRes, err := decodeSAMLAssertion(oktaSAMLRes.RawSAMLAssertion)
	if err != nil {
		return nil, err
	}

	stsRes, err := p.STSClient.AssumeRoleWithSAMLWithContext(ctx, &sts.AssumeRoleWithSAMLInput{
		PrincipalArn:  &decodedSAMLRes.PrincipalARN,
		RoleArn:       &decodedSAMLRes.RoleARN,
		SAMLAssertion: &oktaSAMLRes.RawSAMLAssertion,
	})

	if err != nil {
		return nil, err
	}

	return stsRes.Credentials, nil
}
