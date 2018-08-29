package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/pkg/errors"
)

var nameRegex = regexp.MustCompile(`^[ [:graph:]]*$`)

const apiPrefix = "api/v1"

// Context contains all information needed to use the api functionality.
type Context struct {
	ServerAddress   string
	UserCredentials Credentials
}

// Credentials represents a structure that is able to authenticate a user.
type Credentials interface {
	Apply(header http.Header)
}

// BasicCredentials represents a username and password pair which can be applied to a request.
type BasicCredentials struct {
	Username string
	Password string
}

// Apply applies the given credentials to an HTTP request.
func (cred BasicCredentials) Apply(header http.Header) {
	auth := cred.Username + ":" + cred.Password
	header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
}

// APIKeyCredentials contains an API key which can be applied to a request.
type APIKeyCredentials struct {
	APIKey string
}

// Apply applies the given credentials to an HTTP request.
func (cred APIKeyCredentials) Apply(header http.Header) {
	header.Set(apiKeyHeader, cred.APIKey)
}

// APIErrorResponse is a JSON object that is returned by the ease.ml API when an error occurs.
type APIErrorResponse struct {
	Code      int    `json:"code"`
	Error     string `json:"error"`
	RequestID string `json:"request-id"`
}

func getAPIErrorResponse(resp *http.Response) (*APIErrorResponse, error) {

	var result = &APIErrorResponse{}
	err := json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (resp *APIErrorResponse) String() string {
	if resp == nil {
		return ""
	}
	result := ""
	if resp.Code != 0 {
		result += fmt.Sprintf("%d ", resp.Code)
	}
	if resp.Error != "" {
		result += resp.Error + " "
	}
	if resp.RequestID != "" {
		result += fmt.Sprintf("(Request ID: %s)", resp.RequestID)
	}

	return result
}

func (context Context) sendAPIGetRequest(relPath string, query map[string]string) (resp *http.Response, err error) {

	rawQuery := ""
	if query != nil {
		reqQuery := url.Values{}
		for k, v := range query {
			reqQuery.Set(k, v)
		}
		rawQuery = reqQuery.Encode()
	}

	reqURL := url.URL{
		Scheme:   "http",
		Host:     context.ServerAddress,
		Path:     path.Join(apiPrefix, relPath),
		RawQuery: rawQuery,
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP new request error")
	}
	context.UserCredentials.Apply(req.Header)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP client error")
	}

	if resp.StatusCode != 200 {
		errorResponse, err := getAPIErrorResponse(resp)
		errorString := errorResponse.String()
		if err == nil || errorString == "" {
			errorString = resp.Status
		}
		return nil, errors.New("API error: " + errorString)
	}

	return resp, nil
}

func (context Context) sendAPIPostRequest(relPath string, body io.Reader, contentType string) (resp *http.Response, err error) {

	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, relPath),
	}

	req, err := http.NewRequest("POST", reqURL.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP new request error")
	}
	context.UserCredentials.Apply(req.Header)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP client error")
	}

	if resp.StatusCode != 201 {
		errorResponse, err := getAPIErrorResponse(resp)
		errorString := errorResponse.String()
		if err == nil || errorString == "" {
			errorString = resp.Status
		}
		return nil, errors.New("API error: " + errorString)
	}

	return resp, nil
}

func (context Context) sendAPIPatchRequest(relPath string, body io.Reader, contentType string) (resp *http.Response, err error) {

	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, relPath),
	}

	req, err := http.NewRequest("PATCH", reqURL.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP new request error")
	}
	context.UserCredentials.Apply(req.Header)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP client error")
	}

	if resp.StatusCode != 200 {
		errorResponse, err := getAPIErrorResponse(resp)
		errorString := errorResponse.String()
		if err == nil || errorString == "" {
			errorString = resp.Status
		}
		return nil, errors.New("API error: " + errorString)
	}

	return resp, nil
}
