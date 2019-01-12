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

const defaultSock = "/tmp/execod.sock"

func start() error {
	log.SetLevel(log.DebugLevel)

	sock := os.Getenv("EXECOD_SOCK")
	if sock == "" {
		sock = defaultSock
	}

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
	ln, err := net.Listen("unix", sock)
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
		log.Debugf("received new request")

		cmd := exec.Command(exe, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Infof("running command")
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
	conns := make(chan net.Conn)

	go func() {
		defer close(conns)

		for {
			conn, err := ln.Accept()
			select {
			case <-ctx.Done():
				log.Debugf("listener closed")
				return
			default:
			}

			if err != nil {
				err = errors.Wrap(err, "Accept()")
				log.WithError(err).Error("failed to accept conn")
				continue
			}
			conns <- conn
		}
	}()

	go func() {
		<-ctx.Done()
		log.WithError(ctx.Err()).Errorf("context error")
		err := ln.Close()
		log.WithError(err).Debugf("closed listener")
	}()

	return conns
}
