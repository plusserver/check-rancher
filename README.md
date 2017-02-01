check-rancher - rancher monitoring utility

# Examples

Monitor the state of the "production" environment only

    check-rancher -env production environments

Monitor the hosts of the "production" and "staging" environments only and group host
availability by environments:

    check-rancher -env production,staging -g hosts

Monitor all services labeled with "stage=production" and all system (infrastructure) services

    check-rancher -i stage=production -system services

# Usage

Usage: check-rancher [options] commands...
```
    environments - check status of environments
    hosts        - check hosts (-g groups by environment and uses -w/-c)
    stacks       - check status of stacks
    services     - check status of services
```
Exit code is NRPE compatible (0: OK, 1: warning, 2: critical, 3: unknown)

Options:
```
  -access-key string
    	rancher access key (env RANCHER_ACCESS_KEY)
  -c string
    	critical if less that many percent of resources are available (default "50%")
  -d	debug mode (current unused)
  -e string
    	do not monitor items with these labels (monitor rest). Using both -i and -e is undefined
  -env string
    	limit check to objects in these environments
  -g	group resources, for example all hosts of an environments / all containers of a service
  -h	help
  -i string
    	monitor items with these labels (ignore rest)
  -secret-key string
    	rancher secret key (env RANCHER_SECRET_KEY)
  -system
    	system stacks only / include system services
  -url string
    	rancher url (env RANCHER_URL)
  -v	verbose mode - show status of all checked resources
  -w string
    	warning if less that many percent of resources are available (default "100%")
```

To run the tests, use "go test -p 1"
