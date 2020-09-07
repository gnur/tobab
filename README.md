# tobab
tobab: the poor mans identity aware proxy, easy to use setup for beyondcorp in your homelab

# goals

- Easy to use (single binary with single config file)
- Secure by default (automatic https with letsencrypt, secure cookies)
- Sane defaults (No public access unless explicitly added)
- webui for backend and user management

# non-goals

- Extreme security (no device attestation)
- Reliability (config reloads won't preserve connections as you must kill the server to restart it)



## TODO
- 
- ~~letsencrypt~~
- oidc
- ~~proxy~~
- load conf from env or db
