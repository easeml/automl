package test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"testing"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/ds3lab/easeml/engine/logger"

	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

func createUser(user types.User) (result types.User, err error) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	log := logger.NewProcessLogger(true)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	defer context.Session.Close()

	return context.CreateUser(user)
}

func TestHello(t *testing.T) {
	config := forest.NewConfig("/").Header("X-API-KEY", rootAPIKey)
	r := client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
}

func TestUsersGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response

	var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	// Create users.
	var userBodies = []string{
		fmt.Sprintf(bodyFormat, "user_g_1", "", "active", passwordHash),
		fmt.Sprintf(bodyFormat, "user_g_2", "", "active", passwordHash),
		fmt.Sprintf(bodyFormat, "user_g_3", "", "archived", passwordHash),
	}
	for _, userBody := range userBodies {
		config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey).Body(userBody)
		r = client.POST(t, config)
		forest.ExpectStatus(t, r, 201)
	}

	// Don't authenticate. Get users. Should return 403.
	config = forest.NewConfig("/users")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get users. Should return 401.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all users. Should return 200.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all users. Apply limit. Should return 200.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all users. Order by and order. Should return 200.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "status").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user_g_3", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all users. Apply filter. Should return 200.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Query("status", "archived")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user_g_3", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two users. Should return 200.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "user_g_1,user_g_2")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "user_g_1", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, "user_g_2", forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three users. Verify cursor.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "user_g_1,user_g_2,user_g_3").Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	cursor := forest.JSONPath(t, r, ".metadata.next-page-cursor").(string)
	config.Query("cursor", cursor)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick three users. Verify cursor with ordering.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "status").Query("order", "asc")
	config = config.Query("id", "user_g_1,user_g_2,user_g_3").Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	cursor = forest.JSONPath(t, r, ".metadata.next-page-cursor").(string)
	config.Query("cursor", cursor)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 3, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "user_g_3", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_g_1. Get all users. Should return 200 and only one result.
	config = forest.NewConfig("/users").BasicAuth("user_g_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "user_g_1", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_g_1. Get other users. Should return 200 and empty result.
	config = forest.NewConfig("/users").BasicAuth("user_g_1", password)
	config = config.Query("id", "user_g_2,user_g_3")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.returned-result-size"))
}

func TestUsersPost(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response

	var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	// Don't authenticate. Post user. Should return 403.
	config = forest.NewConfig("/users")
	config = config.Body(fmt.Sprintf(bodyFormat, "user1", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Post user. Should return 401.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey+"_")
	config = config.Body(fmt.Sprintf(bodyFormat, "user1", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Post user. Should return 201.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "user1", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Check that the user exists.
	resourceURL, error := url.Parse(r.Header.Get("Location"))
	assert.Nil(t, error)
	config = forest.NewConfig(resourceURL.Path).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as root. Post user that already exists. Should return 409.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "user1", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 409)

	// Authenticate as user1. Post user. Should return 403.
	config = forest.NewConfig("/users").BasicAuth("user1", password)
	config = config.Body(fmt.Sprintf(bodyFormat, "user2", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 403)
}

func TestUsersGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response

	// Don't authenticate. Get root user. Should return 404.
	config = forest.NewConfig("/users/root")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root user. Should return 401.
	config = forest.NewConfig("/users/root").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get root user. Should return 200.
	config = forest.NewConfig("/users/root").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "root", forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "active", forest.JSONPath(t, r, ".data.status"))

	// Authenticate as root. Get nonexistant user. Should return 404.
	config = forest.NewConfig("/users/nobody").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)
}

func TestUsersPatch(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response

	var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	// Authenticate as root. Post two users. Should return 201.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "user01", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)
	config = config.Body(fmt.Sprintf(bodyFormat, "user02", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Authenticate as root. Patch user name. Should return 200.
	config = forest.NewConfig("/users/user01").Header("X-API-KEY", rootAPIKey)
	config = config.Body(`{"name" : "User 2"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Confirm the changes have been applied.
	config = forest.NewConfig("/users/user01").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "User 2", forest.JSONPath(t, r, ".data.name"))

	// Authenticate as user1. Patch self. Should return 200.
	config = forest.NewConfig("/users/user01").BasicAuth("user01", password)
	config = config.Body(`{"name" : "User 1"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as user1. Patch user2. Should return 404.
	config = forest.NewConfig("/users/user02").BasicAuth("user01", password)
	config = config.Body(`{"name" : "User 2"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 404)
}

func TestUsersLoginLogout(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response

	var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	var password2 = "password2"
	hasher = sha256.New()
	hasher.Write([]byte(password2))
	passwordHash2 := hex.EncodeToString(hasher.Sum(nil))

	// Authenticate as root. Post user. Should return 201.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "user11", "", "active", passwordHash))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Log in as user11 with Basic auth. Should return 200.
	config = forest.NewConfig("/users/login").BasicAuth("user11", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	user11APIKey := r.Header.Get("X-API-KEY")

	// Change password while logged in.
	config = forest.NewConfig("/users/user11").Header("X-API-KEY", user11APIKey)
	config = config.Body(fmt.Sprintf(`{"password" : "%s"}`, passwordHash2))
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Log out as user11 with new password. Should return 200.
	config = forest.NewConfig("/users/logout").BasicAuth("user11", password2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
}

func BenchmarkUserGetById(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	// level := log.GetLevel()
	// log.SetLevel(log.ErrorLevel)
	// defer log.SetLevel(level)

	for n := 0; n < b.N; n++ {
		config = forest.NewConfig("/users/root").Header("X-API-KEY", rootAPIKey)
		r, err = client.Do("GET", config)
		if err != nil {
			panic(err)
		}
		r.Body.Close()
	}
}

func BenchmarkUserLoginLogout(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	// level := log.GetLevel()
	// log.SetLevel(log.PanicLevel)
	// defer log.SetLevel(level)

	// Create a temp user.
	var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))

	// Authenticate as root. Post user. Should return 201, unless the user already exists.
	config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "user_b_1", "", "active", passwordHash))
	r, err = client.Do("POST", config)
	if err != nil {
		panic(err)
	}

	// The code before this moment is not part of the benchmark.
	b.ResetTimer()

	// Main benchmark loop.
	for n := 0; n < b.N; n++ {
		// Login with user id and password and get API key.
		config = forest.NewConfig("/users/login").BasicAuth("user_b_1", password)
		r, err = client.Do("GET", config)
		if err != nil {
			panic(err)
		}
		user11APIKey := r.Header.Get("X-API-KEY")
		if r.StatusCode != 200 {
			panic("status code not 200")
		}
		r.Body.Close()

		// Log out with API key.
		config = forest.NewConfig("/users/logout").Header("X-API-KEY", user11APIKey)
		r, err = client.Do("GET", config)
		if err != nil {
			panic(err)
		}
		if r.StatusCode != 200 {
			panic("status code not 200")
		}
		r.Body.Close()
	}
}

func BenchmarkUsersPost(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	// level := log.GetLevel()
	// log.SetLevel(log.PanicLevel)
	// defer log.SetLevel(level)

	// Main benchmark loop.
	for n := 0; n < b.N; n++ {
		// Create a temp user.
		var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
		var password = "password1"
		hasher := sha256.New()
		hasher.Write([]byte(password))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))

		// Authenticate as root. Post user. Should return 201, unless the user already exists.
		config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
		config = config.Body(fmt.Sprintf(bodyFormat, fmt.Sprintf("user_b_%d", rand.Int()), "", "active", passwordHash))
		r, err = client.Do("POST", config)
		if err != nil {
			panic(err)
		}
		r.Body.Close()
	}
}

type usersResponse struct {
	Data     []types.User             `json:"data"`
	Metadata types.CollectionMetadata `json:"metadata"`
}

func BenchmarkUsersGet1000(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	// level := log.GetLevel()
	// log.SetLevel(log.PanicLevel)
	// defer log.SetLevel(level)

	// Create a 1000 users.
	for n := 0; n < 1000; n++ {
		// Create a temp user.
		var bodyFormat = `{"id" : "%s", "name" : "%s", "status" : "%s", "password" : "%s"}`
		var password = "password1"
		hasher := sha256.New()
		hasher.Write([]byte(password))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))

		// Authenticate as root. Post user. Should return 201, unless the user already exists.
		config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
		config = config.Body(fmt.Sprintf(bodyFormat, fmt.Sprintf("user_b_%d", rand.Int()), "", "active", passwordHash))
		r, err = client.Do("POST", config)
		if err != nil {
			panic(err)
		}
		r.Body.Close()
	}

	// Main benchmark loop.
	for n := 0; n < b.N; n++ {

		// Empty cursor will return the first page of results.
		cursor := ""

		for result := []types.User{}; len(result) < 1000; {

			// Execute a GET response with a cursor and limit.
			config = forest.NewConfig("/users").Header("X-API-KEY", rootAPIKey)
			config = config.Query("limit", 100).Query("cursor", cursor)
			r, err = client.Do("GET", config)
			if err != nil {
				panic(err)
			}

			//Extract results from the response.
			var decodedResponse usersResponse
			err = json.NewDecoder(r.Body).Decode(&decodedResponse)
			if err != nil {
				panic(err)
			}
			cursor = decodedResponse.Metadata.NextPageCursor
			result = append(result, decodedResponse.Data...)
		}
	}
}
