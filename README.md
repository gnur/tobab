# tobab
tobab: the poor mans identity aware proxy, easy to use setup for beyondcorp in your homelab

<img src="./tobab.png" width="350" alt="tobab gopher logo">

It allows you to connect one or more identity providers (currently, only google is supported) and grant access to backends based on the identity of the user.  

# goals

- Easy to use (single binary with single config file)
- Secure by default (automatic https with letsencrypt, secure cookies)
- Sane defaults (No public access unless explicitly added)

# non-goals

- Extreme security
- Reliability (web server restarts whenever a route is added / modified / deleted)

# getting started

- download an appropriate release from the releases page
- place a `tobab.toml` file somewhere and set the env var `TOBAB_CONFIG` var to that location
- configure the google key and secret by creating a new [oauth application](https://developers.google.com/identity/protocols/oauth2/web-server)
- make sure port 80 and port 443 are routed to the host you are running it on
- start tobab with appropriate permissions to bind on port 80 and 443
- ???
- profit

# example config file

```toml
hostname = "login.example.com"
cookiescope = "example.com"
secret = "some-secret"
certdir = "path to dir with write access"
email = "user@example.com"
googlekey = "google id"
googlesecret = "google secret"
loglevel = "debug" #or info, warning, error
databasepath = "./tobab.db"
```

## cli
```
Usage: tobab <command>

Flags:
  -h, --help             Show context-sensitive help.
      --debug
  -c, --config=STRING    config location

Commands:
  run
    start tobab server

  validate
    validate tobab config

  host list
    list all hosts

  host add --hostname=STRING --backend=STRING --type=STRING
    add a new proxy host

  host delete --hostname=STRING
    delete a host

  version
    print tobab version

Run "tobab <command> --help" for more information on a command.
```

### examples
```shell
# add a host
tobab host add --hostname=test.example.com --backend=127.0.0.1:8080 --type=http --public
# list hosts
tobab host list
# delete a host
tobab host delete --hostname=test.example.com
```

## api calls

### example api call to add a route that only allows signed in users with a example.com email address

```http
# @name addHost
POST /v1/api/host
User-Agent: curl/7.64.1
Accept: */*
Cookie: X-Tobab-Token=<token>

{
    "Hostname": "route.example.com",
    "Backend": "https://example.com",
    "Type": "http",
    "Public":false,
    "Globs": [ "*@example.com" ]
}
###
```
### example api call to add a route that allows any signed in user

```http
# @name addHost
POST /v1/api/host
User-Agent: curl/7.64.1
Accept: */*
Cookie: X-Tobab-Token=<token>

{
    "Hostname": "route2.example.com",
    "Backend": "https://example.com",
    "Type": "http",
    "Public":false,
    "Globs": [ "*" ]
}
###
```

### example api call to add a route that allows full access without signing in

```http
# @name addHost
POST /v1/api/host
User-Agent: curl/7.64.1
Accept: */*
Cookie: X-Tobab-Token=<token>

{
    "Hostname": "route2.example.com",
    "Backend": "https://example.com",
    "Type": "http",
    "Public":true,
}
###
```

### example api call to delete a route
```http
# @name delHost
DELETE /v1/api/host/prom.tobab.erwin.land
User-Agent: curl/7.64.1
Accept: */*
Cookie: X-Tobab-Token=<token>
###
```
