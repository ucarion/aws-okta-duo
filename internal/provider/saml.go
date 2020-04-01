package provider

import (
	"encoding/base64"
	"encoding/xml"
	"strings"
)

const awsRoleAttrName = "https://aws.amazon.com/SAML/Attributes/Role"

type decodeSAMLAssertionResult struct {
	PrincipalARN string
	RoleARN      string
}

type samlResponse struct {
	Assertion struct {
		AttributeStatement struct {
			Attributes []struct {
				Name           string `xml:"Name,attr"`
				AttributeValue struct {
					Value string `xml:",innerxml"`
				} `xml:"AttributeValue"`
			} `xml:"Attribute"`
		} `xml:"AttributeStatement"`
	} `xml:"Assertion"`
}

func decodeSAMLAssertion(rawSAMLAssertion string) (decodeSAMLAssertionResult, error) {
	decodedSAMLAssertion, err := base64.StdEncoding.DecodeString(rawSAMLAssertion)
	if err != nil {
		return decodeSAMLAssertionResult{}, err
	}

	var samlResponse samlResponse
	if err := xml.Unmarshal(decodedSAMLAssertion, &samlResponse); err != nil {
		return decodeSAMLAssertionResult{}, err
	}

	for _, attr := range samlResponse.Assertion.AttributeStatement.Attributes {
		if attr.Name == awsRoleAttrName {
			valueParts := strings.Split(attr.AttributeValue.Value, ",")
			return decodeSAMLAssertionResult{PrincipalARN: valueParts[0], RoleARN: valueParts[1]}, nil
		}
	}

	return decodeSAMLAssertionResult{}, nil
}
