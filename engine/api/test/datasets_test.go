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

	log "github.com/sirupsen/logrus"
	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

const testSchemaInSrc1 string = `{
	"nodes":{
		"c1_src":{"singleton":true,"type":"category","class":"class1_src"}
	},"classes":{
		"class1_src":{"dim":16}
	}
}`

const testSchemaOutSrc1 string = `{
	"nodes":{
		"node1_src":{"singleton":true,"type":"tensor","dim":[16]}
	}
}`

func createDataset(dataset model.Dataset) (result model.Dataset, err error) {
	context, err := model.Connect(testDbAddr, testDbName, false)
	if err != nil {
		log.Fatalf("fatal FATAL: %+v", err)
	}
	defer context.Session.Close()
	context.User.ID = dataset.User

	return context.CreateDataset(dataset)
}

func TestDatasetsGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create user_dg_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_dg_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Create datasets.
	var datasets = []model.Dataset{
		model.Dataset{ID: "root/dataset1", User: "root", Name: "Dataset 1", Source: "download", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Dataset{ID: "user2/dataset2", User: "user2", Name: "Dataset 2", Source: "upload", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Dataset{ID: "user2/dataset3", User: "user2", Name: "Dataset 3", Source: "download", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
	}
	for _, dataset := range datasets {
		_, err = createDataset(dataset)
		assert.Nil(t, err)
	}

	// Don't authenticate. Get datasets. Should return 403.
	config = forest.NewConfig("/datasets")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get datasets. Should return 401.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all datasets. Should return 200.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all datasets. Apply limit. Should return 200.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all datasets. Order by and order. Should return 200.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "source").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user2/dataset2", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all datasets. Apply filter. Should return 200.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Query("source", "upload")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user2/dataset2", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two datasets. Should return 200.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "root/dataset1,user2/dataset2")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "root/dataset1", forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, "user2/dataset2", forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three datasets. Verify cursor.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", "root/dataset1,user2/dataset2,user2/dataset3").Query("limit", 2)
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

	// Authenticate as root. Pick three datasets. Verify cursor with ordering.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "source").Query("order", "asc")
	config = config.Query("id", "root/dataset1,user2/dataset2,user2/dataset3").Query("limit", 2)
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
	assert.EqualValues(t, "user2/dataset2", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_dg_1. Get all datasets. Should return 200 and only one result.
	config = forest.NewConfig("/datasets").BasicAuth("user_dg_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "root/dataset1", forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_dg_1. Get other datasets. Should return 200 and empty result.
	config = forest.NewConfig("/datasets").BasicAuth("user_dg_1", password)
	config = config.Query("id", "user2/dataset2,user2/dataset3")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 0, forest.JSONPath(t, r, ".metadata.returned-result-size"))
}

func TestDatasetsPost(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	var bodyFormat = `{"id" : "%s", "name" : "%s", "source" : "%s", "schema-in" : %s, "schema-out" : %s }`
	var testSchemaByteIn1 []byte
	testSchemaByteIn1, err = json.Marshal(testSchemaInSrc1)
	assert.Nil(t, err)
	testSchemaStrIn1 := string(testSchemaByteIn1)
	var testSchemaByteOut1 []byte
	testSchemaByteOut1, err = json.Marshal(testSchemaOutSrc1)
	assert.Nil(t, err)
	testSchemaStrOut1 := string(testSchemaByteOut1)

	// Create user_dp_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_dp_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Post dataset. Should return 403.
	config = forest.NewConfig("/datasets")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset100", "Dataset 1", "upload", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Post dataset. Should return 401.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey+"_")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset100", "Dataset 1", "upload", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Post dataset. Should return 201.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset100", "Dataset 1", "upload", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Check that the dataset exists.
	resourceURL, error := url.Parse(r.Header.Get("Location"))
	assert.Nil(t, error)
	config = forest.NewConfig(resourceURL.Path).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as root. Post dataset that already exists. Should return 409.
	config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset100", "Dataset 1", "upload", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 409)

	// Authenticate as user_dp_1. Post dataset as root. Should return 400.
	config = forest.NewConfig("/datasets").BasicAuth("user_dp_1", password)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset100", "Dataset 1", "upload", testSchemaStrIn1, testSchemaStrOut1))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 400)
}

func TestDatasetsGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create the dataset which we will use in the test.
	_, err = createDataset(model.Dataset{ID: "user2/dataset10", User: "user2", Name: "Dataset 1", Source: "download", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1})
	assert.Nil(t, err)

	// Create user_dbid_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_dbid_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Get root dataset. Should return 404.
	config = forest.NewConfig("/datasets/user2/dataset10")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root dataset. Should return 401.
	config = forest.NewConfig("/datasets/user2/dataset10").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get root dataset. Should return 200.
	config = forest.NewConfig("/datasets/user2/dataset10").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "user2/dataset10", forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "download", forest.JSONPath(t, r, ".data.source"))

	// Authenticate as user_dbid_1. Get dataset of user2. Should return 404.
	config = forest.NewConfig("/datasets/user2/dataset10").BasicAuth("user_dbid_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)
}

func TestDatasetsPatch(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create datasets.
	var datasets = []model.Dataset{
		model.Dataset{ID: "root/dataset10", User: "root", Name: "Dataset 10", Source: "download", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
		model.Dataset{ID: "user_du_1/dataset21", User: "user_du_1", Name: "Dataset 21", Source: "upload", SchemaIn: testSchemaInSrc1, SchemaOut: testSchemaOutSrc1},
	}
	for _, dataset := range datasets {
		_, err = createDataset(dataset)
		assert.Nil(t, err)
	}

	// Create user_du_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(model.User{ID: "user_du_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Authenticate as root. Patch dataset name. Should return 200.
	config = forest.NewConfig("/datasets/root/dataset10").Header("X-API-KEY", rootAPIKey)
	config = config.Body(`{"name" : "Dataset 20 - NEW"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Confirm the changes have been applied.
	config = forest.NewConfig("/datasets/root/dataset10").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, "Dataset 20 - NEW", forest.JSONPath(t, r, ".data.name"))

	// Authenticate as user_du_1. Patch user_du_1/dataset21. Should return 200.
	config = forest.NewConfig("/datasets/user_du_1/dataset21").BasicAuth("user_du_1", password)
	config = config.Body(`{"name" : "Dataset 21 - NEW"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as user_du_1. Patch root/dataset10. Should return 404.
	config = forest.NewConfig("/datasets/root/dataset10").BasicAuth("user_du_1", password)
	config = config.Body(`{"name" : "Dataset 2"}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 404)
}

type datasetsResponse struct {
	Data     []model.Dataset          `json:"data"`
	Metadata model.CollectionMetadata `json:"metadata"`
}

func BenchmarkDatasetsGet1000(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	level := log.GetLevel()
	log.SetLevel(log.PanicLevel)
	defer log.SetLevel(level)

	// Create a 1000 users.
	for n := 0; n < 1000; n++ {
		_, err = createDataset(model.Dataset{
			ID:        fmt.Sprintf("root/%d", rand.Int()),
			User:      "root",
			Name:      "Dataset 1",
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

		for result := []model.Dataset{}; len(result) < 1000; {

			// Execute a GET response with a cursor and limit.
			config = forest.NewConfig("/datasets").Header("X-API-KEY", rootAPIKey)
			config = config.Query("limit", 100).Query("cursor", cursor)
			r, err = client.Do("GET", config)
			if err != nil {
				panic(err)
			}

			//Extract results from the response.
			var decodedResponse datasetsResponse
			err = json.NewDecoder(r.Body).Decode(&decodedResponse)
			if err != nil {
				panic(err)
			}
			cursor = decodedResponse.Metadata.NextPageCursor
			result = append(result, decodedResponse.Data...)
		}
	}
}
