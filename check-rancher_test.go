package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rancher/go-rancher/v2"

	"github.com/stretchr/testify/assert"

	"github.com/braintree/manners"
	"github.com/gorilla/mux"
)

var testcase string

func getClient(t string) (ccc *CheckClientConfig, err error) {

	testcase = t

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/v2-beta{path:.*}", api)

	go func() {
		manners.ListenAndServe("127.0.0.1:8080", router)
	}()

	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       "http://127.0.0.1:8080",
		AccessKey: "blah",
		SecretKey: "bleh",
		Timeout:   5 * time.Second})

	if err != nil {
		panic(err)
	}
	ccc = new(CheckClientConfig)
	ccc.rancher = rancher

	setupCheck(ccc)

	return
}

func done() {
	manners.Close()
}

func api(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	var filename string

	if path == "" {
		if os.Getenv("CHECK_RANCHER_REQUESTLOG") != "" {
			fmt.Printf("%s \"%s\" (%s) => (returning empty answer with schemas header)\n", r.Method, path, r.URL)
		}
		w.Header().Add("X-Api-Schemas", "http://127.0.0.1:8080/v2-beta/schemas")
		w.WriteHeader(http.StatusOK)
	} else {
		if _, err := os.Stat("testcases/" + testcase + path); err == nil {
			filename = "testcases/" + testcase + path
		} else if _, err := os.Stat("testcases/common" + path); err == nil {

			filename = "testcases/common" + path
		} else {
			panic(path + " not found in common and " + testcase)
		}

		if os.Getenv("CHECK_RANCHER_REQUESTLOG") != "" {
			fmt.Printf("%s \"%s\" (%s) => %s\n", r.Method, path, r.URL, filename)
		}

		w.Header().Add("X-Api-Schemas", "http://127.0.0.1:8080/v2-beta/schemas")
		http.ServeFile(w, r, filename)
	}
}

func initTest(testcase string) (ccc *CheckClientConfig) {
	ccc, err := getClient(testcase)

	if err != nil {
		panic(fmt.Sprintf("could not create client: %s", err.Error()))
	}

	if ccc.rancher == nil {
		panic("could not create rancher client")
	}

	if ccc == nil {
		panic("could not create client configuration")
	}

	return
}

func TestIncExcludes(t *testing.T) {
	assert.Equal(t, map[string]string{},
		parseIncExcludes(""))
	assert.Equal(t, map[string]string{"monitorme": "true"},
		parseIncExcludes("monitorme=true"))
	assert.Equal(t, map[string]string{"monitorme": "true", "monitormeaswell": "yes"},
		parseIncExcludes("monitorme=true,monitormeaswell=yes"))
}

func TestFilterLabels(t *testing.T) {
	var ccc CheckClientConfig

	ccc = CheckClientConfig{
		include: map[string]string{"monitorme": "true"},
		exclude: map[string]string{}}

	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"monitorme": "true"}))
	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"monitorme": "true", "rancher.something.something": 17}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"help": "me", "rancher.something": "else"}))

	ccc = CheckClientConfig{
		include: map[string]string{"monitorme": "true", "and": "me"},
		exclude: map[string]string{}}

	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"monitorme": "true"}))
	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"monitorme": "true", "rancher.something.something": 17}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"help": "me", "rancher.something": "else"}))

	ccc = CheckClientConfig{
		include: map[string]string{},
		exclude: map[string]string{"ignoreme": "true"}}

	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"some": "label"}))
	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"some": "label", "more": "labels"}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"ignoreme": "true"}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"ignoreme": "true", "or": "not"}))

	ccc = CheckClientConfig{
		include: map[string]string{},
		exclude: map[string]string{"ignoreme": "true", "and": "me"}}

	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"some": "label"}))
	assert.Equal(t, true, filterLabels(&ccc, map[string]interface{}{"some": "label", "more": "labels"}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"ignoreme": "true"}))
	assert.Equal(t, false, filterLabels(&ccc, map[string]interface{}{"ignoreme": "true", "or": "not"}))

}

func TestEnvironmentsOk(t *testing.T) {
	ccc := initTest("EnvironmentsOk")
	defer done()

	exitCode, _ := checkEnvironments(ccc)
	assert.Equal(t, 0, exitCode)
}

func TestEnvironmentsBroken(t *testing.T) {
	ccc := initTest("EnvironmentsBroken")
	defer done()

	exitCode, alarm := checkEnvironments(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "env emptyenvironment.*unhealthy", alarm)

}

func TestHostsOk(t *testing.T) {
	ccc := initTest("HostsOk")
	defer done()

	exitCode, _ := checkHosts(ccc)

	assert.Equal(t, 0, exitCode)
}

func TestHostsOkTwoEnvs(t *testing.T) {
	ccc := initTest("HostsOkTwoEnvs")
	defer done()

	exitCode, _ := checkHosts(ccc)

	assert.Equal(t, 0, exitCode)
}

func TestHostsNotOkTwoEnvsNotGrouped(t *testing.T) {
	ccc := initTest("HostsNotOkTwoEnvs")
	defer done()

	exitCode, alarm := checkHosts(ccc)

	assert.Equal(t, 1, exitCode)
	assert.Regexp(t, "docker02.*inactive", alarm)
	assert.Regexp(t, "docker03.*inactive", alarm)
}

func TestHostsNotOkTwoEnvsGrouped(t *testing.T) {
	ccc := initTest("HostsNotOkTwoEnvs")
	defer done()

	ccc.groupMode = true
	exitCode, alarm := checkHosts(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "docker02.*inactive", alarm)
	assert.Regexp(t, "docker03.*inactive", alarm)
	assert.Regexp(t, "Default: 1 of 3", alarm)
}

func TestStacksOk(t *testing.T) {
	ccc := initTest("StacksOk")
	defer done()

	exitCode, _ := checkStacks(ccc)

	assert.Equal(t, 0, exitCode)
}

func TestStacksDegraded(t *testing.T) {
	ccc := initTest("StacksDegraded")
	defer done()

	exitCode, alarm := checkStacks(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "worlddominationapp.*Default.*degraded", alarm)
}

func TestStacksHealthcheckFailing(t *testing.T) {
	ccc := initTest("StacksHealthcheckFailing")
	defer done()

	exitCode, alarm := checkStacks(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "worlddominationapp.*Default.*degraded", alarm)
}

func TestServicesOk(t *testing.T) {
	ccc := initTest("ServicesOk")
	defer done()

	exitCode, _ := checkServices(ccc)

	assert.Equal(t, 0, exitCode)
}

func TestServicesBrokenWithoutLabels(t *testing.T) {
	ccc := initTest("ServicesBroken")
	defer done()

	exitCode, alarm := checkServices(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "donotmonitorme/broken", alarm)
	assert.Regexp(t, "monitorme/nothingeverworks", alarm)
}

func TestServicesBrokenWithInclude(t *testing.T) {
	ccc := initTest("ServicesBroken")
	ccc.include = map[string]string{"monitor": "true"}
	defer done()

	exitCode, alarm := checkServices(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "monitorme/nothingeverworks", alarm)
}

func TestServicesBrokenWithExclude(t *testing.T) {
	ccc := initTest("ServicesBroken")
	ccc.exclude = map[string]string{"monitor": "false"}
	defer done()

	exitCode, alarm := checkServices(ccc)

	assert.Equal(t, 2, exitCode)
	assert.Regexp(t, "monitorme/nothingeverworks", alarm)
}
