package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gorilla/mux"
)

func main() {
	cfg := Config{
		CertDir: "/home/erwin/certs",
		Hosts: []Host{
			{
				Host:    "tobab.erwin.land",
				Backend: "https://postman-echo.com",
				Type:    "http",
			},
			{
				Host:    "echo.tobab.erwin.land",
				Backend: "http://httpbin.org",
				Type:    "http",
			},
			{
				Host:    "ip.tobab.erwin.land",
				Backend: "https://ifconfig.co",
				Type:    "http",
			},
		},
		Email:   "test@dekeijzer.xyz",
		Staging: true,
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = cfg.Email

	if cfg.Staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}

	r := mux.NewRouter()
	var hosts []string

	for _, conf := range cfg.Hosts {
		proxy, err := generateProxy(conf)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.Host(conf.Host).PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})
		hosts = append(hosts, conf.Host)
	}

	//log.Fatal(autotls.RunWithManager(r, &m))
	err := certmagic.HTTPS(hosts, r)
	if err != nil {
		fmt.Println(err)
	}
}

func generateProxy(host Host) (http.Handler, error) {
	url, err := url.Parse(host.Backend)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{Director: func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", url.Hostname())
		req.Header.Add("X-Origin-Host", host.Host)
		req.Host = url.Host

		url.Path = req.URL.Path
		req.URL = url

		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(req.URL)
			fmt.Println(string(dump))
		}

	}, Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}}

	return proxy, nil
}
