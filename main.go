package main

import (
	"bytes"
	"context"
	"flag"
	"net"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func main() {
	cfgFile := flag.String("config", "config.toml", "Config file")
	flag.Parse()

	var cfg Config
	_, err := toml.DecodeFile(*cfgFile, &cfg)
	if err != nil {
		panic(err)
	}

	auth, err := getAuth(cfg)
	if err != nil {
		panic(err)
	}

	callback, err := getKnownHosts(cfg)
	if err != nil {
		panic(err)
	}

	wrappedCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// fmt.Println(hostname, remote, key)
		_ = callback(hostname, remote, key)
		return nil
	}

	client, err := goph.NewConn(&goph.Config{
		User:     cfg.Remote.User,
		Addr:     cfg.Remote.Host,
		Port:     cfg.Remote.Port,
		Auth:     auth,
		Timeout:  goph.DefaultTimeout,
		Callback: wrappedCallback,
	})
	if err != nil {
		panic(err)
	}

	rootContext, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	var wg sync.WaitGroup

	for i := range cfg.Tail {
		cfg.Tail[i].Run(rootContext, &wg, client)
	}

	for i := range cfg.Pipe {
		cfg.Pipe[i].Run(rootContext, &wg, client)
	}

	wg.Wait()
}

func getAuth(config Config) (goph.Auth, error) {
	if config.Remote.Agent {
		return goph.UseAgent()
	}

	return goph.Key(config.Remote.PrivateKeyFile, "")
}

func getKnownHosts(config Config) (ssh.HostKeyCallback, error) {
	if config.Remote.KnownHosts {
		return goph.DefaultKnownHosts()
	}

	if config.Remote.KnownHost != "" {
		knownKey, err := ssh.ParsePublicKey([]byte(config.Remote.KnownHost))
		if err != nil {
			return nil, err
		}

		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if bytes.Equal(knownKey.Marshal(), key.Marshal()) {
				return nil
			}
			spew.Dump(key)
			return &knownhosts.KeyError{}
		}, nil
	}

	return ssh.InsecureIgnoreHostKey(), nil
}
