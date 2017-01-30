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
	debugMode, verboseMode, groupMode bool
	warning, critical                 float64
	rancher                           *client.RancherClient
}

func main() {

	var ccc CheckClientConfig
	setupCheck(&ccc)

	var warningPercent, criticalPercent, includeStr, excludeStr string

	flag.StringVar(&ccc.rancherURL, "url", os.Getenv("RANCHER_URL"), "rancher url (env RANCHER_URL)")
	flag.StringVar(&ccc.accessKey, "access-key", os.Getenv("RANCHER_ACCESS_KEY"), "rancher access key (env RANCHER_ACCESS_KEY)")
	flag.StringVar(&ccc.secretKey, "secret-key", os.Getenv("RANCHER_SECRET_KEY"), "rancher secret key (env RANCHER_SECRET_KEY)")
	flag.BoolVar(&ccc.verboseMode, "v", false, "verbose mode - show status of all checked resources")
	flag.BoolVar(&ccc.debugMode, "d", false, "debug mode (current unused)")
	flag.BoolVar(&helpMode, "h", false, "help")
	flag.BoolVar(&ccc.groupMode, "g", false, "group resources, for example all hosts of an environments / all containers of a service")
	flag.StringVar(&warningPercent, "w", "100%", "warning if less that many percent of resources are available")
	flag.StringVar(&criticalPercent, "c", "50%", "critical if less that many percent of resources are available")
	flag.StringVar(&includeStr, "i", "", "monitor items with these labels (ignore rest)")
	flag.StringVar(&excludeStr, "e", "", "do not monitor items with these labels (monitor rest)")

	flag.Parse()

	if len(ccc.rancherURL) < 1 {
		fmt.Println("need rancher URL")
		os.Exit(2)
	}

	if len(ccc.accessKey) < 1 || len(ccc.secretKey) < 1 {
		fmt.Println("need access key / secret key")
		os.Exit(2)
	}

	if len(warningPercent) > 0 {
		var w int
		fmt.Sscanf(warningPercent, "%d%%", &w)
		ccc.warning = float64(w) / float64(100)
	} else {
		ccc.warning = 0
	}

	if len(criticalPercent) > 0 {
		var c int
		fmt.Sscanf(criticalPercent, "%d%%", &c)
		ccc.critical = float64(c) / float64(100)
	} else {
		ccc.critical = 0
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
		Url:       ccc.rancherURL,
		AccessKey: ccc.accessKey,
		SecretKey: ccc.secretKey,
		Timeout:   10 * time.Second})
	if err != nil {
		panic(err)
	}
	if rancher == nil {
		panic("did not get a rancher client")
	}

	ccc.rancher = rancher

	var e int
	var alarm string

	switch args[0] {
	case "all":
		cmdAll(&ccc)
	case "environments":
		e, alarm = checkEnvironments(&ccc)
	case "hosts":
		e, alarm = checkHosts(&ccc)
	case "stacks":
		e, alarm = checkStacks(&ccc)
	case "services":
		e, alarm = checkServices(&ccc)
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

func setupCheck(ccc *CheckClientConfig) {

	// defaults relevant in tests
	ccc.warning = 1
	ccc.critical = 0.5

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

func cmdAll(ccc *CheckClientConfig) {}

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

func checkEnvironments(ccc *CheckClientConfig) (e int, alarm string) {
	rancher := ccc.rancher
	environments, err := rancher.Project.List(nil)
	if err != nil {
		panic(err)
	}

	e = 0

	for _, env := range environments.Data {
		envAlarm := fmt.Sprintf("env %s running %s with %d active hosts is %s/%s", env.Name, env.Orchestration, len(env.Members), env.State, env.HealthState)
		if ccc.verboseMode {
			fmt.Println(envAlarm)
		}

		if env.State != "active" || env.HealthState != "healthy" {
			e = 2
			alarm = alarm + envAlarm + " "
		}
	}

	return
}

func checkHosts(ccc *CheckClientConfig) (e int, alarm string) {
	rancher := ccc.rancher
	
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

		if ccc.verboseMode {
			fmt.Printf("%s(%s) in env %s is %s\n", host.Hostname, host.AgentIpAddress, environ.Name, host.State)
		}
	}

	if ccc.groupMode {
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
			if availRate < ccc.critical {
				alarm = alarm + fmt.Sprintf("%s: %s: %s ", environ, availStr, alarm2)
				e = 2
			} else if availRate < ccc.warning {
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
		if availRate < ccc.critical {
			alarm = alarm + " " + availStr
			e = 2
		} else if availRate < ccc.warning {
			alarm = alarm + " " + availStr
			if e == 0 {
				e = 1
			}
		}
	}

	return
}

func checkStacks(ccc *CheckClientConfig) (e int, alarm string) {
	rancher := ccc.rancher
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

		if ccc.verboseMode {
			fmt.Printf("%s in env %s is %s/%s\n", stack.Name, env.Name, stack.State, stack.HealthState)
		}

		if stack.State != "active" || stack.HealthState != "healthy" {
			alarm = alarm + fmt.Sprintf("%s in env %s (%s/%s) ", stack.Name, env.Name, stack.State, stack.HealthState)
			e = 2
		}
	}

	return
}

func checkServices(ccc *CheckClientConfig) (e int, alarm string) {
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

		if ccc.verboseMode {
			fmt.Printf("env=%s, stack=%s, currentscale=%d health=%s, name=%s, scale=%d, stackid=%s, state=%s, transition=%s, transitionmessage=%s, transitioningprogress=%d\n", env.Name, stack.Name, service.CurrentScale, service.HealthState, service.Name, service.Scale, service.StackId, service.State, service.Transitioning, service.TransitioningMessage, service.TransitioningProgress)
		}
	}

	return
}
