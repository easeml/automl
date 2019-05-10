package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"github.com/ds3lab/easeml/client/go/client/types"

	"github.com/pkg/errors"
)

const apiKeyHeader = "X-API-KEY"

// Login takes a username and password and attempts to log the user in. If the login was
// successful, the API key is returned which can be used to authenticate the user.
func (context Context) Login(username, password string) (result string, err error) {
	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, "users/login"),
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return "", errors.Wrap(err, "HTTP new request error")
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "HTTP client error")
	}

	if resp.StatusCode != 200 {
		errorResponse, err := getAPIErrorResponse(resp)
		errorString := errorResponse.String()
		if err == nil || errorString == "" {
			errorString = resp.Status
		}
		return "", errors.New("API error: " + resp.Status)
	}

	apiKey := resp.Header.Get(apiKeyHeader)

	return apiKey, nil
}

// Logout takes the provided user credentials and tries to log the user out.
func (context Context) Logout() error {

	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, "users/logout"),
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, "HTTP new request error")
	}
	context.UserCredentials.Apply(req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "HTTP client error")
	}

	if resp.StatusCode != 200 {
		errorResponse, err := getAPIErrorResponse(resp)
		errorString := errorResponse.String()
		if err == nil || errorString == "" {
			errorString = resp.Status
		}
		return errors.New("API error: " + resp.Status)
	}

	return nil
}

// GetUsers returns all users from the service.
func (context Context) GetUsers(status string) (result []types.User, err error) {

	result = []types.User{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if status != "" {
			query["status"] = status
		}
		resp, err := context.sendAPIGetRequest("users", query)
		if err != nil {
			return nil, err
		}

		type getUsersResponse struct {
			Data     []types.User             `json:"data"`
			Metadata types.CollectionMetadata `json:"metadata"`
			Links    map[string]string        `json:"links"`
		}
		respObject := getUsersResponse{}
		err = json.NewDecoder(resp.Body).Decode(&respObject)
		if err != nil {
			return nil, errors.Wrap(err, "JSON decode error")
		}
		nextCursor = respObject.Metadata.NextPageCursor
		result = append(result, respObject.Data...)

		if nextCursor == "" || len(respObject.Data) == 0 {
			break
		}
	}

	return result, nil
}

// GetUserByID returns a user given its ID.
func (context Context) GetUserByID(id string) (result *types.User, err error) {

	if id == "" {
		id = types.UserThis
	}

	resp, err := context.sendAPIGetRequest(path.Join("users", id), nil)
	if err != nil {
		return nil, err
	}

	type getUserByIDResponse struct {
		Data types.User `json:"data"`
	}
	respObject := getUserByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return nil, errors.Wrap(err, "JSON decode error")
	}

	return &respObject.Data, nil
}

// GetMyID returns the ID of the current user.
func (context Context) GetMyID() (result string, err error) {

	resp, err := context.sendAPIGetRequest(path.Join("users", types.UserThis), nil)
	if err != nil {
		return "", err
	}

	type getUserByIDResponse struct {
		Data types.User `json:"data"`
	}
	respObject := getUserByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return "", errors.Wrap(err, "JSON decode error")
	}

	return respObject.Data.ID, nil
}

// CreateUser creates a new user given the provided parameters.
func (context Context) CreateUser(id, password, name string) (string, error) {

	if id == "" {
		panic("id argument cannot be empty")
	}
	if password == "" {
		panic("password argument cannot be empty")
	}
	if nameRegex.MatchString(name) == false {
		return "", errors.New("invalid user name")
	}

	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	user := types.User{
		ID:           id,
		Name:         name,
		PasswordHash: passwordHash,
	}

	userBytes, err := json.Marshal(&user)
	if err != nil {
		return "", err
	}
	resp, err := context.sendAPIPostRequest("users", bytes.NewReader(userBytes), "application/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return id, nil
}
