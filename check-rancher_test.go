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

func getClient(t string) (rancher *client.RancherClient, err error) {
	testcase = t

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/v2-beta{path:.*}", api)

	go func() {
		manners.ListenAndServe("127.0.0.1:8080", router)
	}()

	rancher, err = client.NewRancherClient(&client.ClientOpts{
		Url:       "http://127.0.0.1:8080",
		AccessKey: "blah",
		SecretKey: "bleh",
		Timeout:   5 * time.Second})

	return
}

func done() {
	manners.Close()
}

func api(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	var filename string

	if path == "" {
		// fmt.Printf("%s \"%s\" (%s) => (returning empty answer with schemas header)\n", r.Method, path, r.URL)
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

		// fmt.Printf("%s \"%s\" (%s) => %s\n", r.Method, path, r.URL, filename)

		w.Header().Add("X-Api-Schemas", "http://127.0.0.1:8080/v2-beta/schemas")
		http.ServeFile(w, r, filename)
	}
}

func TestGetClient(t *testing.T) {
	assert := assert.New(t)

	rancher, err := getClient("common")
	defer done()

	if err != nil {
		assert.Nil(err, fmt.Sprintf("could not create rancher instance: %s", err.Error()))
	}

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")
}

func TestEnvironmentsOk(t *testing.T) {
	assert := assert.New(t)

	rancher, err := getClient("EnvironmentsOk")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, _ := checkEnvironments(rancher)

		assert.Equal(0, exitCode)
	}
}

func TestEnvironmentsBroken(t *testing.T) {
	assert := assert.New(t)

	rancher, err := getClient("EnvironmentsBroken")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, alarm := checkEnvironments(rancher)

		assert.Equal(2, exitCode)
		assert.Regexp("env emptyenvironment.*unhealthy", alarm)
	}

}

func TestHostsOk(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("HostsOk")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, _ := checkHosts(rancher)

		assert.Equal(0, exitCode)
	}
}

func TestHostsOkTwoEnvs(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("HostsOkTwoEnvs")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, _ := checkHosts(rancher)

		assert.Equal(0, exitCode)
	}
}

func TestHostsNotOkTwoEnvsNotGrouped(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("HostsNotOkTwoEnvs")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, alarm := checkHosts(rancher)

		assert.Equal(1, exitCode)
		assert.Regexp("docker02.*inactive", alarm)
		assert.Regexp("docker03.*inactive", alarm)
	}
}

func TestHostsNotOkTwoEnvsGrouped(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	groupMode = true
	rancher, err := getClient("HostsNotOkTwoEnvs")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, alarm := checkHosts(rancher)

		assert.Equal(2, exitCode)
		assert.Regexp("docker02.*inactive", alarm)
		assert.Regexp("docker03.*inactive", alarm)
		assert.Regexp("Default: 1 of 3", alarm)
	}
}


func TestStacksOk(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("StacksOk")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, _ := checkStacks(rancher)

		assert.Equal(0, exitCode)
	}
}

func TestStacksDegraded(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("StacksDegraded")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, alarm := checkStacks(rancher)

		assert.Equal(2, exitCode)
		assert.Regexp("worlddominationapp.*Default.*degraded", alarm)
	}
}

func TestStacksHealthcheckFailing(t *testing.T) {
	assert := assert.New(t)

	setupCheck()
	rancher, err := getClient("StacksHealthcheckFailing")
	defer done()

	assert.Nil(err)
	assert.NotNil(rancher, "we were supposed to get a rancher client")

	if rancher != nil {
		exitCode, alarm := checkStacks(rancher)

		assert.Equal(2, exitCode)
		assert.Regexp("worlddominationapp.*Default.*degraded", alarm)
	}
}

