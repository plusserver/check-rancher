check-rancher - rancher monitoring utility

Usage: check-rancher [options] commands...
```
    environments - check status of environments
    hosts        - check hosts (-g groups by environment and uses -w/-c)
    stacks       - check status of stacks
```
Exit code is NRPE compatible (0: OK, 1: warning, 2: critical, 3: unknown)

Options:
```
  -access-key string
    	rancher access key (env RANCHER_ACCESS_KEY)
  -c string
    	critical if less that many percent of resources are available (default "50%")
  -d	debug mode (current unused)
  -g	group resources, for example all hosts of an environments / all containers of a service
  -h	help
  -secret-key string
    	rancher secret key (env RANCHER_SECRET_KEY)
  -url string
    	rancher url (env RANCHER_URL)
  -v	verbose mode - show status of all checked resources
  -w string
    	warning if less that many percent of resources are available (default "100%")
```
