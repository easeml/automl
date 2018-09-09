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

	"github.com/ds3lab/easeml/database/model"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

func createModule(module model.Module) (result model.Module, err error) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	if err != nil {
		log.Fatalf("fatal: %+v", err)
	}
	defer context.Session.Close()
	context.User.ID = module.User

	return context.CreateModule(module)
}

func TestModulesGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create user_mg_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_mg_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Create modules.
	var modules = []model.Module{
		model.Module{ID: "root/module1", User: "root", Name: "Module 1", Source: "download", Type: "model", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Module{ID: "user2/module2", User: "user2", Name: "Module 2", Source: "upload", Type: "model", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Module{ID: "user2/module3", User: "user2", Name: "Module 3", Source: "download", Type: "objective", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
	}
	for _, module := range modules {
		_, err = createModule(module)
		assert.Nil(t, err)
	}

	// Don't authenticate. Get modules. Should return 403.
	config = forest.NewConfig("/modules")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get modules. Should return 401.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all modules. Should return 200.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all modules. Apply limit. Should return 200.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all modules. Order by and order. Should return 200.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "source").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user2/module2", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all modules. Apply filter. Should return 200.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Query("source", "upload")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user2/module2", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two modules. Should return 200.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "root/module1,user2/module2")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "root/module1", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, "user2/module2", forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three modules. Verify cursor.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "root/module1,user2/module2,user2/module3").Query("limit", 2)
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

	// Authenticate as root. Pick three modules. Verify cursor with ordering.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "source").Query("order", "asc")
	config = config.Query("id", "root/module1,user2/module2,user2/module3").Query("limit", 2)
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
	assert.EqualValues(t, "user2/module2", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_mg_1. Get all modules. Should return 200 and only one result.
	config = forest.NewConfig("/modules").BasicAuth("user_mg_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "root/module1", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_mg_1. Get other modules. Should return 200 and empty result.
	config = forest.NewConfig("/modules").BasicAuth("user_mg_1", password)
	config = config.Query("id", "user2/module2,user2/module3")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.returned-result-size"))
}

func TestModulesPost(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	var bodyFormat = `{"id" : "%s", "name" : "%s", "source" : "%s", "type" : "%s", "schema-in" : %s, "schema-out" : %s }`
	var testSchemaByteIn1 []byte
	testSchemaByteIn1, err = json.Marshal(testSchemaInSrc1)
	assert.Nil(t, err)
	testSchemaStrIn1 := string(testSchemaByteIn1)
	var testSchemaByteOut1 []byte
	testSchemaByteOut1, err = json.Marshal(testSchemaOutSrc1)
	assert.Nil(t, err)
	testSchemaStrOut1 := string(testSchemaByteOut1)

	// Create user_mp_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_mp_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Post module. Should return 403.
	config = forest.NewConfig("/modules")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/module100", "Module 1", "upload", "model", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Post module. Should return 401.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey+"_")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/module100", "Module 1", "upload", "model", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Post module. Should return 201.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/module100", "Module 1", "upload", "model", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Check that the module exists.
	resourceURL, error := url.Parse(r.Header.Get("Location"))
	assert.Nil(t, error)
	config = forest.NewConfig(resourceURL.Path).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as root. Post module that already exists. Should return 409.
	config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/module100", "Module 1", "upload", "model", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 409)

	// Authenticate as user_mp_1. Post module as root. Should return 400.
	config = forest.NewConfig("/modules").BasicAuth("user_mp_1", password)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/module100", "Module 1", "upload", "model", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 400)
}

func TestModulesGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create the module which we will use in the test.
	_, err = createModule(model.Module{ID: "user2/module10", User: "user2", Name: "Module 1", Source: "download", Type: "model", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1})
	assert.Nil(t, err)

	// Create user_mbid_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_mbid_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Get root module. Should return 404.
	config = forest.NewConfig("/modules/user2/module10")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root module. Should return 401.
	config = forest.NewConfig("/modules/user2/module10").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get root module. Should return 200.
	config = forest.NewConfig("/modules/user2/module10").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "user2/module10", forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "download", forest.JSONPath(t, r, ".data.source"))

	// Authenticate as user_mbid_1. Get module of user2. Should return 404.
	config = forest.NewConfig("/modules/user2/module10").BasicAuth("user_mbid_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)
}

func TestModulesPatch(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create modules.
	var modules = []model.Module{
		model.Module{ID: "root/module10", User: "root", Name: "Module 10", Source: "download", Type: "model", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Module{ID: "user_mu_1/module21", User: "user_mu_1", Name: "Module 21", Source: "upload", Type: "model", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
	}
	for _, module := range modules {
		_, err = createModule(module)
		assert.Nil(t, err)
	}

	// Create user_mu_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_mu_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Authenticate as root. Patch module name. Should return 200.
	config = forest.NewConfig("/modules/root/module10").Header("X-API-KEY", rootAPIKey)
	config = config.Body(`{"name" : "Module 20 - NEW"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Confirm the changes have been applied.
	config = forest.NewConfig("/modules/root/module10").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "Module 20 - NEW", forest.JSONPath(t, r, ".data.name"))

	// Authenticate as user_mu_1. Patch user_mu_1/module21. Should return 200.
	config = forest.NewConfig("/modules/user_mu_1/module21").BasicAuth("user_mu_1", password)
	config = config.Body(`{"name" : "Module 21 - NEW"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as user_mu_1. Patch root/module10. Should return 404.
	config = forest.NewConfig("/modules/root/module10").BasicAuth("user_mu_1", password)
	config = config.Body(`{"name" : "Module 2"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 404)
}

type modulesResponse struct {
	Data     []model.Module           `json:"data"`
	Metadata model.CollectionMetadata `json:"metadata"`
}

func BenchmarkModulesGet1000(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	level := log.GetLevel()
	log.SetLevel(log.PanicLevel)
	defer log.SetLevel(level)

	// Create a 1000 users.
	for n := 0; n < 1000; n++ {
		_, err = createModule(model.Module{
			ID:        fmt.Sprintf("root/%d", rand.Int()),
			User:      "root",
			Name:      "Module 1",
			Source:    "download",
			SchemaIn:  testSchemaInSrc1,
			SchemaOut: testSchemaOutSrc1,
		})
		if err != nil {
			panic(err)
		}
	}

	// Main benchmark loop.
	for n := 0; n < b.N; n++ {

		// Empty cursor will return the first page of results.
		cursor := ""

		for result := []model.Module{}; len(result) < 1000; {

			// Execute a GET response with a cursor and limit.
			config = forest.NewConfig("/modules").Header("X-API-KEY", rootAPIKey)
			config = config.Query("limit", 100).Query("cursor", cursor)
			r, err = client.Do("GET", config)
			if err != nil {
				panic(err)
			}

			//Extract results from the response.
			var decodedResponse modulesResponse
			err = json.NewDecoder(r.Body).Decode(&decodedResponse)
			if err != nil {
				panic(err)
			}
			cursor = decodedResponse.Metadata.NextPageCursor
			result = append(result, decodedResponse.Data...)
		}
	}
}
