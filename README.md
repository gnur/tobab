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
