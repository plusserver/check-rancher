package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rancher/go-rancher/v2"
)

var helpMode bool

var environmentCache map[string]*client.Project
var stackCache map[string]*client.Stack

type CheckClientConfig struct {
	rancherURL, accessKey, secretKey  string
	rancher                           *client.RancherClient
	stack, service, host, environment string
}

func main() {

	var ccc CheckClientConfig
	setupCheck(&ccc)

	var checkType string // unused, for compatibility with rancher-icinga

	flag.StringVar(&ccc.rancherURL, "url", os.Getenv("RANCHER_URL"), "rancher url (env RANCHER_URL)")
	flag.StringVar(&ccc.accessKey, "access-key", os.Getenv("RANCHER_ACCESS_KEY"), "rancher access key (env RANCHER_ACCESS_KEY)")
	flag.StringVar(&ccc.secretKey, "secret-key", os.Getenv("RANCHER_SECRET_KEY"), "rancher secret key (env RANCHER_SECRET_KEY)")
	flag.BoolVar(&helpMode, "h", false, "help")
	flag.StringVar(&ccc.environment, "env", "Default", "limit check to objects in this environment")
	flag.StringVar(&checkType, "type", "", "(ignored)")
	flag.StringVar(&ccc.stack, "stack", "", "stack name to check (requires env)")
	flag.StringVar(&ccc.service, "service", "", "service name to check (requires env and stack")
	flag.StringVar(&ccc.host, "host", "", "host to check")

	flag.Parse()

	if helpMode {
		usage()
		return
	}

	if len(ccc.rancherURL) < 1 {
		fmt.Println("need rancher URL")
		os.Exit(2)
	}

	if len(ccc.accessKey) < 1 || len(ccc.secretKey) < 1 {
		fmt.Println("need access key / secret key")
		os.Exit(2)
	}

	args := flag.Args()

	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       ccc.rancherURL,
		AccessKey: ccc.accessKey,
		SecretKey: ccc.secretKey,
		Timeout:   10 * time.Second})

	if err != nil {
		fmt.Println("CRITICAL: Cannot connect to rancher server:", err)
		os.Exit(2)
	}

	ccc.rancher = rancher

	var e int
	var alarm string

	if len(args) != 0 {
		usage()
		return
	}

	if ccc.stack != "" && ccc.environment != "" && ccc.service == "" {
		e, alarm = checkStack(&ccc)
	} else if ccc.service != "" && ccc.stack != "" && ccc.environment != "" {
		e, alarm = checkService(&ccc)
	} else if ccc.host != "" {
		e, alarm = checkHost(&ccc)
	} else {
		usage()
		return
	}

	if e == 0 {
		fmt.Println("OK:", alarm)
	} else if e == 1 {
		fmt.Println("WARNING:", alarm)
	} else if e == 2 {
		fmt.Println("CRITICAL:", alarm)
	} else {
		fmt.Printf("UNKNOWN (%d): %s\n", e, alarm)
	}

	os.Exit(e)
}

func setupCheck(ccc *CheckClientConfig) {

	environmentCache = make(map[string]*client.Project)
	stackCache = make(map[string]*client.Stack)
}

func usage() {
	fmt.Println(
		`check-rancher - rancher monitoring utility

Usage: check-rancher [options]...

	check-rancher -host my.rancher.agent.com
	check-rancher -env PROD -stack mystack
	check-rancher -env DEV -stack ipsec -service ipsec
    
Exit code is NRPE compatible (0: OK, 1: warning, 2: critical, 3: unknown)

Options:
`)
	flag.PrintDefaults()
}

func debugOutput(something interface{}) {
	out, _ := json.MarshalIndent(something, "", "  ")
	fmt.Println(string(out))
}

func getEnvironment(rancher *client.RancherClient, id string) (env *client.Project, err error) {
	var ok bool
	if env, ok = environmentCache[id]; !ok {
		env, err = rancher.Project.ById(id)
		if err == nil {
			environmentCache[id] = env
		}
	}
	return
}

func getStack(rancher *client.RancherClient, id string) (st *client.Stack, err error) {
	var ok bool
	if st, ok = stackCache[id]; !ok {
		st, err = rancher.Stack.ById(id)
		if err == nil {
			stackCache[id] = st
		}
	}
	return
}

func checkStack(ccc *CheckClientConfig) (int, string) {
	rancher := ccc.rancher

	stacks, err := rancher.Stack.List(nil)

	if err != nil {
		panic(err)
	}

	for _, stack := range stacks.Data {

		env, err := getEnvironment(rancher, stack.AccountId)

		if err != nil {
			panic(err)
		}

		if stack.Name == ccc.stack && env.Name == ccc.environment {
			summary := fmt.Sprintf("stack %s in environment %s is %s and %s", ccc.stack, env.Name, stack.State, stack.HealthState)
			
			if err != nil {
				panic(err)
			}

			if stack.State == "active" && stack.HealthState == "healthy" {
				return 0, summary
			} else {
				return 2, summary
			}
		}
	}

	return 2, "stack " + ccc.stack + " not found in environment " + ccc.environment
}

func checkService(ccc *CheckClientConfig) (int, string) {
	rancher := ccc.rancher

	services, err := rancher.Service.List(nil)

	if err != nil {
		panic(err)
	}

	for _, service := range services.Data {

		stack, err := getStack(rancher, service.StackId)

		if err != nil {
			panic(err)
		}

		env, err := getEnvironment(rancher, stack.AccountId)

		if err != nil {
			panic(err)
		}

		if stack.Name == ccc.stack && service.Name == ccc.service && env.Name == ccc.environment {
			summary := fmt.Sprintf("service %s/%s in environment %s is %s ", stack.Name, service.Name, env.Name, service.HealthState)
			
			if service.HealthState == "healthy" {
				return 0, summary
			} else {
				return 2, summary
			}
		}
	}

	return 2, "service " + ccc.stack + "/" + ccc.service + " not found in environment " + ccc.environment
}

func checkHost(ccc *CheckClientConfig) (e int, alarm string) {
	rancher := ccc.rancher

	hosts, err := rancher.Host.List(nil)
	if err != nil {
		panic(err)
	}

	for _, host := range hosts.Data {
		if host.Hostname == ccc.host {
			if host.State == "active" {
				return 0, "host " + ccc.host + " is " + host.State
			} else if host.State == "inactive" {
				return 1, "host " + ccc.host + " is " + host.State + " (this is probably intentional)"
			} else {
				return 2, "host " + ccc.host + " is " + host.State
			}
		}
	}

	return 3, "host " + ccc.host + " not found"
}
