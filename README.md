# tobab
tobab: an opinionated poor mans identity-aware proxy, easy to use setup for beyondcorp in your homelab

<img src="./tobab.png" width="350" alt="tobab gopher logo">

It uses passkeys for simple and robust authentication.

## goals

- Easy to use (single binary with single config file)
- Secure by default (automatic https with letsencrypt, secure cookies)
- Sane defaults (No public access unless explicitly added)

## non-goals

- Extreme security
- Reliability (web server restarts whenever a route is added / modified / deleted)
- Customization
- Pretty

## wishlist (not implemented yet)

- docker integration (use the docker API to determine containers to route traffic into)
- docker builds
- full integration test suite that can run every night
- admin UI that shows all seen users, shows routes and allows you to edit routes
- metrics

## getting started

- download an appropriate release from the releases page
- place a `tobab.toml` file somewhere and set the env var `TOBAB_CONFIG` var to that location
- make sure port 80 and port 443 are routed to the host you are running it on
- start tobab with appropriate permissions to bind on port 80 and 443
- add routes using the CLI or the API
- ???
- profit

# example config file

```toml
hostname = "login.example.com" #hostname where the login occurs
cookiescope = "example.com"
secret = "some-secret"
salt = ""
certdir = "path to dir with write access"
email = "user@example.com"
loglevel = "debug" #or info, warning, error
databasepath = "./tobab.db"
adminglobs = [ "*@example.com" ] #globs of email addresses that are allowed to use the admin API
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
# add a host to listen on test.example.com that proxies all requests to 127.0.0.1:8080
# please be aware, if you add a host that isn't public it should have the same suffix as the cookie scope!
tobab host add --hostname=test.example.com --backend=http://127.0.0.1:8080 --type=http --public
# list hosts
tobab host list
# delete a host
tobab host delete --hostname=test.example.com
```

## api calls



# acknowledgements

This project could hot have been what it is today without these great libraries:

 - github.com/gorilla/mux excellent light weight request router
 - github.com/markbates/goth library that handles all third party authentication stuffs
 - github.com/caddyserver/certmagic letsencrypt made very, very easy
 - github.com/sirupsen/logrus logging library that is perfect
 - github.com/asdine/storm embedded database built upon bolt which makes persistence very easy

 # alternatives

 - Combine github.com/traefik/traefik with a forward auth provider like github.com/gnur/beyondauth or github.com/thomseddon/traefik-forward-auth
 - Combine github.com/oauth2-proxy/oauth2-proxy with some kind of certificate maintenance service like github.com/certbot/certbot
