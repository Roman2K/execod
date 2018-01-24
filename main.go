package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := start(); err != nil {
		log.WithError(errors.Wrap(err, "start()")).Fatal("main error")
	}
}

func start() error {
	// Command
	if len(os.Args) < 2 {
		return errors.New("usage: cmd [arg ...]")
	}
	var (
		exe  = os.Args[1]
		args = os.Args[2:]
	)
	log.Infof("command: %q %q", exe, args)

	// Listen
	ln, err := net.Listen("unix", "/tmp/execod.sock")
	if err != nil {
		return errors.Wrap(err, "net.Listen()")
	}
	defer ln.Close()
	log.WithField("addr", ln.Addr()).Infof("listening")

	// Clean stop on SIGINT
	ctx, cancel := interruptContext()
	defer cancel()

	for conn := range listen(ctx, ln) {
		conn.Close()
		log.Infof("received new request")

		cmd := exec.Command(exe, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.WithField("cmd", cmd).Infof("running command")
		t0 := time.Now()
		err = cmd.Run()
		if err != nil {
			log.WithError(errors.Wrap(err, "Run()")).Error("command failed")
			continue
		}
		log.WithField("runtime", time.Since(t0)).Infof("command run successfully")
	}
	return nil
}

func interruptContext() (ctx context.Context, cancel func()) {
	ctx, cancel = context.WithCancel(context.Background())
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		defer cancel()
		<-sigs
		signal.Stop(sigs)
	}()
	return
}

func listen(ctx context.Context, ln net.Listener) <-chan net.Conn {
	var (
		conns = make(chan net.Conn)
		done  = make(chan struct{})
	)

	go func() {
		defer close(conns)

		for {
			conn, err := ln.Accept()
			select {
			case <-done:
				log.Infof("listener closed")
				return
			default:
			}

			if err != nil {
				log.WithError(errors.Wrap(err, "Accept()")).Error("failed to accept conn")
				continue
			}
			conns <- conn
		}
	}()

	go func() {
		defer close(done)
		<-ctx.Done()
		log.WithError(ctx.Err()).Errorf("context error")
		err := ln.Close()
		log.WithError(err).Infof("closed listener")
	}()

	return conns
}
