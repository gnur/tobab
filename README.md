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
- Reliability (config reloads won't preserve connections as you must kill the server to restart it)

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

[hosts."echo.example.com"]
backend = "https://httpbin.org"
type = "http"
public = true

[hosts."ip.example.com"]
backend = "https://ifconfig.co"
type = "http"
allowedglobs = [ "everyone" ]

[hosts."admin.example.com"]
backend = "http://localhost:8080"
type = "http"
allowedglobs = [ "admin" ]

[globs]
admin = "*@example.com"
everyone = "*"
```

In this example, the difference between `ip.example.com` and `echo.example.com` is that `echo.example.com` can be used without signing in to any identity provider. But to visit `ip.example.com` you need to be signed in, but anyone can use it once you are signed in.

In the globs definition a `*` can be any amount of characters, including none at all. In the case of the above admin group any `@example.com` email will be allowed access to `admin.example.com`.
