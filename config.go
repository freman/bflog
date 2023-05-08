package main

import (
	"context"
	"os"
	"sync"

	"github.com/melbahja/goph"
)

type Config struct {
	Remote RemoteConfig
	Tail   []TailConfig
	Pipe   []PipeConfig
}

type RemoteConfig struct {
	Host           string
	Port           uint
	User           string
	Agent          bool
	PrivateKeyFile string
	KnownHosts     bool
	KnownHost      string
}

type TailConfig struct {
	Output  string
	Src     string
	Disable bool
}

type PipeConfig struct {
	Output  string
	Cmd     []string
	Disable bool
}

type Commander interface { // todo abstract further to not be goph.cmd dependant
	Command(name string, args ...string) (*goph.Cmd, error)
}

func (t TailConfig) Run(ctx context.Context, wg *sync.WaitGroup, c Commander) error {
	if t.Disable {
		return nil
	}

	fn, err := os.Create(t.Output)
	if err != nil {
		return err
	}

	cmd, err := c.Command("/bin/tail", "-f", t.Src)
	if err != nil {
		fn.Close()
		return err
	}

	cmd.Context = ctx

	cmd.Stdout = fn
	cmd.Stderr = fn

	if err := cmd.Start(); err != nil {
		fn.Close()
		return err
	}

	wg.Add(1)

	go func() {
		if err := cmd.Wait(); err != nil {
			panic(err)
		}

		fn.Close()
		wg.Done()
	}()

	return nil
}

func (t PipeConfig) Run(ctx context.Context, wg *sync.WaitGroup, c Commander) error {
	if t.Disable {
		return nil
	}

	fn, err := os.Create(t.Output)
	if err != nil {
		return err
	}

	cmd, err := c.Command(t.Cmd[0], t.Cmd[1:]...)
	if err != nil {
		fn.Close()

		return err
	}

	cmd.Context = ctx

	cmd.Stdout = fn
	cmd.Stderr = fn

	if err := cmd.Start(); err != nil {
		fn.Close()
		return err
	}

	wg.Add(1)

	go func() {
		if err := cmd.Wait(); err != nil {
			panic(err)
		}

		fn.Close()
		wg.Done()
	}()

	return nil
}
