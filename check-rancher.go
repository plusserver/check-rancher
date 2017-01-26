package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rancher/go-rancher/v2"
)

var rancherURL, accessKey, secretKey string
var debugMode, helpMode, verboseMode, groupMode bool
var warning, critical float64

var environmentCache map[string]*client.Project
var stackCache map[string]*client.Stack

func main() {

	setupCheck()

	var warningPercent, criticalPercent string

	flag.StringVar(&rancherURL, "url", os.Getenv("RANCHER_URL"), "rancher url (env RANCHER_URL)")
	flag.StringVar(&accessKey, "access-key", os.Getenv("RANCHER_ACCESS_KEY"), "rancher access key (env RANCHER_ACCESS_KEY)")
	flag.StringVar(&secretKey, "secret-key", os.Getenv("RANCHER_SECRET_KEY"), "rancher secret key (env RANCHER_SECRET_KEY)")
	flag.BoolVar(&verboseMode, "v", false, "verbose mode - show status of all checked resources")
	flag.BoolVar(&debugMode, "d", false, "debug mode (current unused)")
	flag.BoolVar(&helpMode, "h", false, "help")
	flag.BoolVar(&groupMode, "g", false, "group resources, for example all hosts of an environments / all containers of a service")
	flag.StringVar(&warningPercent, "w", "100%", "warning if less that many percent of resources are available")
	flag.StringVar(&criticalPercent, "c", "50%", "critical if less that many percent of resources are available")

	flag.Parse()

	if len(rancherURL) < 1 {
		fmt.Println("need rancher URL")
		os.Exit(2)
	}

	if len(accessKey) < 1 || len(secretKey) < 1 {
		fmt.Println("need access key / secret key")
		os.Exit(2)
	}

	if len(warningPercent) > 0 {
		var w int
		fmt.Sscanf(warningPercent, "%d%%", &w)
		warning = float64(w) / float64(100)
	} else {
		warning = 0
	}

	if len(criticalPercent) > 0 {
		var c int
		fmt.Sscanf(criticalPercent, "%d%%", &c)
		critical = float64(c) / float64(100)
	} else {
		critical = 0
	}

	args := flag.Args()

	if len(args) == 0 {
		usage()
		return
	}

	if helpMode {
		usage()
		return
	}

	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       rancherURL,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Timeout:   10 * time.Second})
	if err != nil {
		panic(err)
	}
	if rancher == nil {
		panic("rancher karp0tt")
	}

	var e int
	var alarm string

	switch args[0] {
	case "all":
		cmdAll(rancher)
	case "environments":
		e, alarm = checkEnvironments(rancher)
	case "hosts":
		e, alarm = checkHosts(rancher)
	case "stacks":
		e, alarm = checkStacks(rancher)
	case "services":
		e, alarm = checkServices(rancher)
	default:
		usage()
		return
	}

	if e == 0 {
		fmt.Println("OK")
	} else if e == 1 {
		fmt.Println("WARNING:", alarm)
	} else if e == 2 {
		fmt.Println("CRITICAL:", alarm)
	} else {
		fmt.Printf("UNKNOWN (%d): %s\n", e, alarm)
	}

	os.Exit(e)
}

func setupCheck() {
	
	// defaults relevant in tests
	warning = 1
	critical = 0.5
	
	environmentCache = make(map[string]*client.Project)
	stackCache = make(map[string]*client.Stack)
}

func usage() {
	fmt.Println(
		`check-rancher - rancher monitoring utility

Usage: check-rancher [options] commands...

    environments - check status of environments
    hosts        - check hosts (-g groups by environment and uses -w/-c)
    stacks       - check status of stacks

Exit code is NRPE compatible (0: OK, 1: warning, 2: critical, 3: unknown)

Options:
`)
	flag.PrintDefaults()
}

func cmdAll(rancher *client.RancherClient) {}

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

func checkEnvironments(rancher *client.RancherClient) (e int, alarm string) {
	environments, err := rancher.Project.List(nil)
	if err != nil {
		panic(err)
	}

	e = 0

	for _, env := range environments.Data {
		envAlarm := fmt.Sprintf("env %s running %s with %d active hosts is %s/%s", env.Name, env.Orchestration, len(env.Members), env.State, env.HealthState)
		if verboseMode {
			fmt.Println(envAlarm)
		}

		if env.State != "active" || env.HealthState != "healthy" {
			e = 2
			alarm = alarm + envAlarm + " "
		}
	}

	return
}

func checkHosts(rancher *client.RancherClient) (e int, alarm string) {
	hosts, err := rancher.Host.List(nil)
	if err != nil {
		panic(err)
	}

	groups := make(map[string][]client.Host)

	for _, host := range hosts.Data {
		environ, err := getEnvironment(rancher, host.AccountId)
		if err != nil {
			panic(err)
		}

		groups[environ.Name] = append(groups[environ.Name], host)

		if verboseMode {
			fmt.Printf("%s(%s) in env %s is %s\n", host.Hostname, host.AgentIpAddress, environ.Name, host.State)
		}
	}

	if groupMode {
		var alarm2 string
		for environ, hosts := range groups {
			var avail int = 0
			for _, host := range hosts {
				if host.State != "active" {
					alarm2 = alarm2 + fmt.Sprintf("%s is %s ", host.Hostname, host.State)
				} else {
					avail = avail + 1
				}
			}
			availRate := float64(avail) / float64(len(hosts))
			availStr := fmt.Sprintf("%d of %d hosts available", avail, len(hosts))
			if availRate < critical {
				alarm = alarm + fmt.Sprintf("%s: %s: %s ", environ, availStr, alarm2)
				e = 2
			} else if availRate < warning {
				alarm = alarm + fmt.Sprintf("%s: %s: %s ", environ, availStr, alarm2)
				if e == 0 {
					e = 1
				}
			}
		}
	} else {
		var avail int = 0
		for _, host := range hosts.Data {
			if host.State != "active" {
				alarm = alarm + fmt.Sprintf("%s is %s ", host.Hostname, host.State)
			} else {
				avail = avail + 1
			}
		}
		availRate := float64(avail) / float64(len(hosts.Data))
		availStr := fmt.Sprintf("%d of %d hosts available", avail, len(hosts.Data))
		if availRate < critical {
			alarm = alarm + " " + availStr
			e = 2
		} else if availRate < warning {
			alarm = alarm + " " + availStr
			if e == 0 {
				e = 1
			}
		}
	}

	return
}

func checkStacks(rancher *client.RancherClient) (e int, alarm string) {
	stacks, err := rancher.Stack.List(nil)
	if err != nil {
		panic(err)
	}

	e = 0

	for _, stack := range stacks.Data {
		env, err := getEnvironment(rancher, stack.AccountId)
		if err != nil {
			panic(err)
		}

		if verboseMode {
			fmt.Printf("%s in env %s is %s/%s\n", stack.Name, env.Name, stack.State, stack.HealthState)
		}

		if stack.State != "active" || stack.HealthState != "healthy" {
			alarm = alarm + fmt.Sprintf("%s in env %s (%s/%s) ", stack.Name, env.Name, stack.State, stack.HealthState)
			e = 2
		}
	}

	return
}

func checkServices(rancher *client.RancherClient) (e int, alarm string) {
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

		if verboseMode {
			fmt.Printf("env=%s, stack=%s, currentscale=%d health=%s, name=%s, scale=%d, stackid=%s, state=%s, transition=%s, transitionmessage=%s, transitioningprogress=%d\n", env.Name, stack.Name, service.CurrentScale, service.HealthState, service.Name, service.Scale, service.StackId, service.State, service.Transitioning, service.TransitioningMessage, service.TransitioningProgress)
		}
	}

	return
}
