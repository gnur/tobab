# tobab
tobab: an opinionated poor mans identity-aware proxy enabler. Use it as a forward auth target with caddy, nginx or traefik.

<img src="./tobab.png" width="350" alt="tobab gopher logo">

It uses passkeys for simple and robust authentication.

## goals

- Passkey enabled user management
- Admin with Web UI for access management
- Easy to use (single docker container with simple config)

## non-goals

- any authn that isn't passkeys

## wishlist (not implemented yet)

- metrics
- API key support for non-browser session based validation
- access denied message
- better error handling with feedback to user
- better splitting of templates and javascript (not a single script for login and register)
- testing with Traefik
- testing with nginx
- additional storage interface implementations to allow it to be more cloud native

## getting started

- See the `k8s-example` dir for a kustomize setup for tobab and deploy to k8s
- make sure dns is setup correctly
- Setup caddy to use this new endpoint for forward auth:
```
login.example.com {
  reverse_proxy tobab.tabab.svc

}
secure.example.com {
        forward_auth tobab.tobab.svc {
                uri /verify
        }
        reverse_proxy some_other_host:8080
}
```
- create a new user at `login.example.com/register` (first user created becomes the admin user)
- visit `secure.example.com` and be authenticated through your passkey
- login with the new user




# example config file

```toml
hostname = "login.example.com" #hostname where the login occurs
displayname = "example displayname" #used for passkey creation
cookiescope = "example.com" #this will allow all subdomains of example.com to have sso with tobab
loglevel = "debug" #or info, warning, error
databasepath = "./tobab.db"
```


# acknowledgements

This project could hot have been what it is today without these great libraries:

 - github.com/gin-gonic/gin excellent request router
 - github.com/sirupsen/logrus logging library that is perfect
 - github.com/asdine/storm embedded database built upon bolt which makes persistence very easy
