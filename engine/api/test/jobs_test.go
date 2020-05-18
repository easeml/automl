package test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/ds3lab/easeml/engine/database/model"
	"github.com/ds3lab/easeml/engine/database/model/types"
	"github.com/ds3lab/easeml/engine/logger"

	"github.com/emicklei/forest"
	"github.com/stretchr/testify/assert"
)

func createJob(job types.Job, status string) (result types.Job, err error) {

	context, err := model.Connect(testDbAddr, testDbName, false)
	log := logger.NewProcessLogger(true)
	if err != nil {
		log.WriteFatal(fmt.Sprintf("fatal: %+v", err))
	}
	defer context.Session.Close()
	context.User.ID = job.User

	result, err = context.CreateJob(job)
	if err != nil {
		return
	}

	return context.UpdateJob(result.ID, model.F{"status": status})
}

func TestJobsGet(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create user_jg_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(types.User{ID: "user_jg_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Create jobs.
	var jobs = []types.Job{
		types.Job{User: "root", Dataset: "root/dataset1", Models: []string{"root/model1"}, Objective: "root/objective1"},
		types.Job{User: "root", Dataset: "root/dataset3", Models: []string{"root/model1"}, Objective: "root/objective1"},
		types.Job{User: "user1", Dataset: "root/dataset2", Models: []string{"root/model1"}, Objective: "root/objective1"},
	}
	for i := range jobs {
		jobs[i], err = createJob(jobs[i], "running")
		assert.Nil(t, err)
	}

	// Don't authenticate. Get jobs. Should return 403.
	config = forest.NewConfig("/jobs")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Get jobs. Should return 401.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get all jobs. Should return 200.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.total-result-size"), "Field total-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.returned-result-size"), "Field returned-result-size empty.")
	assert.NotNil(t, forest.JSONPath(t, r, ".metadata.next-page-cursor"), "Field next-page-cursor empty.")

	// Authenticate as root. Get all jobs. Apply limit. Should return 200.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey).Query("limit", 2)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Get all jobs. Order by and order. Should return 200.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "dataset").Query("order", "desc")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, jobs[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as root. Get all jobs. Apply filter. Should return 200.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Query("user", "user1")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, jobs[2].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, 1, forest.JSONPath(t, r, ".metadata.returned-result-size"))

	// Authenticate as root. Pick two jobs. Should return 200.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s", jobs[1].ID.Hex(), jobs[2].ID.Hex()))
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, jobs[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))
	assert.EqualValues(t, jobs[2].ID.Hex(), forest.JSONPath(t, r, ".data.1.id"))

	// Authenticate as root. Pick three jobs. Verify cursor.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", jobs[0].ID.Hex(), jobs[1].ID.Hex(), jobs[2].ID.Hex())).Query("limit", 2)
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

	// Authenticate as root. Pick three jobs. Verify cursor with ordering.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Query("order-by", "dataset").Query("order", "asc")
	config = config.Query("id", fmt.Sprintf("%s,%s,%s", jobs[0].ID.Hex(), jobs[1].ID.Hex(), jobs[2].ID.Hex())).Query("limit", 2)
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
	assert.EqualValues(t, jobs[1].ID.Hex(), forest.JSONPath(t, r, ".data.0.id"))

	// Authenticate as user_jg_1. Get all jobs. Should return 200 and only jobs by root.
	config = forest.NewConfig("/jobs").BasicAuth("user_jg_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.total-result-size"))
	assert.EqualValues(t, 2, forest.JSONPath(t, r, ".metadata.returned-result-size"))
	assert.EqualValues(t, "root", forest.JSONPath(t, r, ".data.0.user"))
	assert.EqualValues(t, "root", forest.JSONPath(t, r, ".data.1.user"))
}

func TestJobsPost(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error
	var bodyFormat = `{"dataset" : "%s", "models" : ["%s"], "objective": "%s" }`

	// Create user_jp_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(types.User{ID: "user_jp_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Post job. Should return 403.
	config = forest.NewConfig("/jobs")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset1", "root/model1", "root/objective1"))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 403)

	// Use wrong API key. Post job. Should return 401.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey+"_")
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset1", "root/model1", "root/objective1"))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Post job. Should return 201.
	config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset1", "root/model1", "root/objective1"))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Check that the job exists.
	var resourceURL *url.URL
	resourceURL, err = url.Parse(r.Header.Get("Location"))
	assert.Nil(t, err)
	config = forest.NewConfig(resourceURL.Path).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "root", forest.JSONPath(t, r, ".data.user"))

	// Authenticate as user_jp_1. Post job as root. Should return 400.
	config = forest.NewConfig("/jobs").BasicAuth("user_jp_1", password)
	config = config.Body(fmt.Sprintf(bodyFormat, "root/dataset1", "root/model1", "root/objective1"))
	r = client.POST(t, config)
	forest.ExpectStatus(t, r, 201)

	// Check that the job exists.
	resourceURL, err = url.Parse(r.Header.Get("Location"))
	assert.Nil(t, err)
	config = forest.NewConfig(resourceURL.Path).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.EqualValues(t, "user_jp_1", forest.JSONPath(t, r, ".data.user"))
}

func TestJobsGetById(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error
	var job types.Job

	// Create the job which we will use in the test.
	job, err = createJob(types.Job{User: "user1", Dataset: "root/dataset1", Models: []string{"root/model1"}, Objective: "root/objective1"}, "running")
	assert.Nil(t, err)

	// Create user_jbid_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(types.User{ID: "user_jbid_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Don't authenticate. Get root job. Should return 404.
	config = forest.NewConfig("/jobs/" + job.ID.Hex())
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)

	// Use wrong API key. Get root job. Should return 401.
	config = forest.NewConfig("/jobs/"+job.ID.Hex()).Header("X-API-KEY", rootAPIKey+"_")
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 401)

	// Authenticate as root. Get root job. Should return 200.
	config = forest.NewConfig("/jobs/"+job.ID.Hex()).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, job.ID.Hex(), forest.JSONPath(t, r, ".data.id"))
	assert.Equal(t, "root/dataset1", forest.JSONPath(t, r, ".data.dataset"))

	// Authenticate as user_jbid_1. Get job of user1. Should return 404.
	config = forest.NewConfig("/jobs/"+job.ID.Hex()).BasicAuth("user_jbid_1", password)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 404)
}

func TestJobsPatch(t *testing.T) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Create jobs.
	var jobs = []types.Job{
		types.Job{User: "root", Dataset: "root/dataset1", Models: []string{"root/model1"}, Objective: "root/objective1"},
		types.Job{User: "user_ju_1", Dataset: "root/dataset1", Models: []string{"root/model1"}, Objective: "root/objective1"},
	}
	for i := range jobs {
		jobs[i], err = createJob(jobs[i], "running")
		assert.Nil(t, err)
	}

	// Create user_ju_1.
	var password = "password1"
	hasher := sha256.New()
	hasher.Write([]byte(password))
	passwordHash := hex.EncodeToString(hasher.Sum(nil))
	_, err = createUser(types.User{ID: "user_ju_1", PasswordHash: passwordHash, Status: "active"})
	assert.Nil(t, err)

	// Authenticate as root. Patch job name. Should return 200.
	config = forest.NewConfig("/jobs/"+jobs[0].ID.Hex()).Header("X-API-KEY", rootAPIKey)
	config = config.Body(`{"accept-new-models" : true}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Confirm the changes have been applied.
	config = forest.NewConfig("/jobs/"+jobs[0].ID.Hex()).Header("X-API-KEY", rootAPIKey)
	r = client.GET(t, config)
	forest.ExpectStatus(t, r, 200)
	assert.Equal(t, true, forest.JSONPath(t, r, ".data.accept-new-models"))

	// Authenticate as user_ju_1. Patch job that belongs to self. Should return 200.
	config = forest.NewConfig("/jobs/"+jobs[1].ID.Hex()).BasicAuth("user_ju_1", password)
	config = config.Body(`{"accept-new-models" : true}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 200)

	// Authenticate as user_ju_1. Patch job that belongs to root. Should return 401.
	config = forest.NewConfig("/jobs/"+jobs[0].ID.Hex()).BasicAuth("user_ju_1", password)
	config = config.Body(`{"accept-new-models" : true}`)
	r = client.PATCH(t, config)
	forest.ExpectStatus(t, r, 401)
}

type jobsResponse struct {
	Data     []types.Job              `json:"data"`
	Metadata types.CollectionMetadata `json:"metadata"`
}

func BenchmarkJobsGet1000(b *testing.B) {
	var config *forest.RequestConfig
	var r *http.Response
	var err error

	// Temporarily turn off logging.
	// level := log.GetLevel()
	// log.SetLevel(log.PanicLevel)
	// defer log.SetLevel(level)

	// Create a 1000 users.
	for n := 0; n < 1000; n++ {
		_, err = createJob(types.Job{User: "root", Dataset: "root/dataset1", Models: []string{"root/model1"}, Objective: "root/objective1"}, "running")

		if err != nil {
			panic(err)
		}
	}

	// Main benchmark loop.
	for n := 0; n < b.N; n++ {

		// Empty cursor will return the first page of results.
		cursor := ""

		for result := []types.Job{}; len(result) < 1000; {

			// Execute a GET response with a cursor and limit.
			config = forest.NewConfig("/jobs").Header("X-API-KEY", rootAPIKey)
			config = config.Query("limit", 100).Query("cursor", cursor)
			r, err = client.Do("GET", config)
			if err != nil {
				panic(err)
			}

			//Extract results from the response.
			var decodedResponse jobsResponse
			err = json.NewDecoder(r.Body).Decode(&decodedResponse)
			if err != nil {
				panic(err)
			}
			cursor = decodedResponse.Metadata.NextPageCursor
			result = append(result, decodedResponse.Data...)
		}
	}
}
