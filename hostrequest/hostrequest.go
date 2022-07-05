package hostrequest

import (
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt"
	gonnect "github.com/sumeet70/atlas-gonnect"
	atlasjwt "github.com/sumeet70/atlas-gonnect/atlas-jwt"
	atlasoauth2 "github.com/sumeet70/atlas-gonnect/atlas-oauth2"
	"github.com/sumeet70/atlas-gonnect/store"
)

type HostRequest struct {
	Addon     *gonnect.Addon
	ClientKey string
	Tenant    *store.AtlassianHost
}

func FromRequest(r *http.Request) (*HostRequest, error) {
	ihttpClient := r.Context().Value("httpClient")
	if ihttpClient == nil {
		return nil, errors.New("Could not get httpClient from request context; no httpClient")
	}
	httpClient, ok := ihttpClient.(*HostRequest)
	if !ok {
		return nil, errors.New("Could not get httpClient from request context; couldn't cast pointer")
	}

	// We do the request to the database here
	// We could also do it for every new request, but I think this shouldn't be
	// required
	// however, technically this could lead to difficulties if the secret changes
	tenant, err := httpClient.Addon.Store.Get(httpClient.ClientKey)
	if err != nil {
		return nil, err
	}
	httpClient.Tenant = tenant

	return httpClient, nil
}

func (h HostRequest) modifyRequest(req *http.Request) (*http.Request, error) {
	baseUrl, err := url.Parse(h.Tenant.BaseURL)
	if err != nil {
		return nil, err
	}
	if req.URL.Host == "" {
		req.URL.Host = baseUrl.Host
		req.URL.Scheme = baseUrl.Scheme
		req.URL.Path = filepath.Join(baseUrl.Path, req.URL.Path)
	}
	return req, nil
}

func (h HostRequest) AsAddon(req *http.Request) (*http.Request, error) {
	// The qsh must only read contain the path after /wiki/
	// We therefore generate the claims first and prepend the baseUrl later
	claims := struct {
		QueryStringHash string `json:"qsh"`
		jwt.StandardClaims
	}{
		QueryStringHash: atlasjwt.CreateQueryStringHash(req, false, ""),
		StandardClaims: jwt.StandardClaims{
			Issuer:    *h.Addon.Key,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(3 * time.Minute).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(h.Tenant.SharedSecret))
	if err != nil {
		return nil, err
	}

	h.modifyRequest(req)

	req.Header.Set("Authorization", "JWT "+signedToken)
	// TODO: User-Agent

	return req, nil
}

func (h HostRequest) AsUser(req *http.Request, accountId string) (*http.Request, error) {
	iScopes := h.Addon.AddonDescriptor["scopes"].([]interface{})
	scopes := make([]string, len(iScopes))
	for idx, val := range iScopes {
		scopes[idx] = val.(string)
	}
	token, err := atlasoauth2.GetAccessToken(h.Tenant, accountId, scopes)
	if err != nil {
		return nil, err
	}
	h.modifyRequest(req)
	req.Header.Set("Authorization", "Bearer "+token)
	// TODO: User-Agent
	return req, nil
}
