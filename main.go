package main

import (
	"flag"
	"fmt"
	humanize "github.com/dustin/go-humanize"
	"github.com/jlaffaye/ftp"
	"io"
	"io/ioutil"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

func checkError(err error) bool {
	if err == nil {
		return false
	}
	if strings.HasPrefix(err.Error(), "150") {
		return false
	}
	if strings.HasPrefix(err.Error(), "250") {
		return false
	}
	return true
}

func StressReads(con *ftp.ServerConn, files []string, iters int) (int64, error) {
	var nread int64
	for i := 0; i < iters; i++ {
		fi := files[rand.Intn(len(files))]
		r, err := con.Retr(fi)
		if checkError(err) {
			fmt.Println("err: ", err)
			return nread, err
		}
		n, err := io.Copy(ioutil.Discard, r)
		if checkError(err) {
			fmt.Println("err: ", err)
			return nread, err
		}

		nread += n

		err = r.Close()
		if checkError(err) {
			fmt.Println("err: ", err)
			return nread, err
		}
	}
	return nread, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	host := flag.String("host", "localhost", "hostname of ftp server")
	port := flag.Int("port", 21, "port of ftp server")
	threads := flag.Int("threads", 1, "number of concurrent threads to run")
	user := flag.String("user", "test-user", "username")
	pass := flag.String("pass", "password", "password")
	iters := flag.Int("iter", 1, "number of iterations per thread")
	rfile := flag.String("file", "", "number of iterations per thread")

	flag.Parse()

	con, err := ftp.Connect(fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Println(err)
		return
	}

	err = con.Login(*user, *pass)
	if err != nil {
		fmt.Println(err)
		return
	}

	if *rfile == "" {
		fmt.Println("Please specify a file to read with `-file=X`")
		return
	}

	// temp
	files := []string{*rfile}

	donech := make(chan int64)
	starttime := time.Now()
	for i := 0; i < *threads; i++ {
		go func() {
			n, err := StressReads(con, files, *iters)
			if err != nil {
				fmt.Println(err)
			}
			donech <- n
		}()
	}

	var sum int64
	for i := 0; i < *threads; i++ {
		sum += <-donech
		fmt.Println("sum = ", sum)
	}
	end := time.Now().Sub(starttime)
	fmt.Printf("Read a total of %s\n", humanize.Bytes(uint64(sum)))
	fmt.Printf("speed = %s/s\n", humanize.Bytes(uint64(float64(sum)/end.Seconds())))
}
