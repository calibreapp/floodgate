package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"golang.org/x/net/context"

	"time"

	"io"

	"strings"

	socks5 "github.com/armon/go-socks5"
	"github.com/aybabtme/iocontrol"
)

var (
	rpool   *iocontrol.ReaderPool
	wpool   *iocontrol.WriterPool
	latency time.Duration
)

func main() {
	var listen, ratekpbs, latencyms string

	flag.StringVar(&listen, "listen", "", "address to listen on, e.g 127.0.0.1:8000 (required)")
	flag.StringVar(&ratekpbs, "rate", "", "rate to limit to, in kpbs (required)")
	flag.StringVar(&latencyms, "latency", "", "Latency to add to operations, in ms (required)")
	flag.Parse()

	verrs := []string{}
	if listen == "" {
		verrs = append(verrs, "listen is a required flag")
	}
	if ratekpbs == "" {
		verrs = append(verrs, "rate is a required flag")
	}
	if latencyms == "" {
		verrs = append(verrs, "latency is a required flag")
	}
	rate, err := strconv.Atoi(ratekpbs)
	if err != nil {
		verrs = append(verrs, "ratekbps needs to be numeric only")
	}
	latencyi, err := strconv.Atoi(latencyms)
	if err != nil {
		verrs = append(verrs, "latency needs to be numeric only")
	}
	latency = time.Duration(latencyi) * time.Millisecond

	if len(verrs) > 0 {
		flag.Usage()
		fmt.Println("")
		fmt.Println(strings.Join(verrs, "\n"))
		os.Exit(2)
	}

	rpool = iocontrol.NewReaderPool(rate*iocontrol.KiB, 10*time.Millisecond)
	wpool = iocontrol.NewWriterPool(rate*iocontrol.KiB, 10*time.Millisecond)

	conf := &socks5.Config{
		Dial: dial,
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}
	if err := server.ListenAndServe("tcp", listen); err != nil {
		panic(err)
	}
}

type conn struct {
	net.Conn
	r    io.Reader
	relr func()
	w    io.Writer
	relw func()
}

func (c *conn) Read(b []byte) (n int, err error) {
	time.Sleep(latency)
	return c.r.Read(b)
}
func (c *conn) Write(b []byte) (n int, err error) {
	time.Sleep(latency)
	return c.w.Write(b)
}

func (c *conn) Close() error {
	c.relr()
	c.relw()
	return c.Conn.Close()
}

func dial(ctx context.Context, network, addr string) (net.Conn, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	r, relr := rpool.Get(c)
	w, relw := wpool.Get(c)
	return &conn{
		Conn: c,
		r:    r,
		relr: relr,
		w:    w,
		relw: relw,
	}, nil
}
