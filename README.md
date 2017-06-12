check-rancher - rancher monitoring utility

Checks the state of rancher resources by name. For use with rancher-icinga.



# Examples

Check state of a rancher agent

    check-rancher -host my.rancher.agent.com

Check state of a stack

    check-rancher -env PROD -stack mystack

Check state of a service

    check-rancher -env DEV -stack ipsec -service ipsec

# Usage

```
check-rancher - rancher monitoring utility

Usage: check-rancher [options]...

        check-rancher -host my.rancher.agent.com
        check-rancher -env PROD -stack mystack
        check-rancher -env DEV -stack ipsec -service ipsec
    
Exit code is NRPE compatible (0: OK, 1: warning, 2: critical, 3: unknown)

Options:

  -access-key string
        rancher access key (env RANCHER_ACCESS_KEY)
  -env string
        limit check to objects in this environment (default "Default")
  -h    help
  -host string
        host to check
  -secret-key string
        rancher secret key (env RANCHER_SECRET_KEY)
  -service string
        service name to check (requires env and stack
  -stack string
        stack name to check (requires env)
  -type string
        (ignored)
  -url string
        rancher url (env RANCHER_URL)
```
