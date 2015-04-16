package main

import (
	"bufio"
	"flag"
	"fmt"
	humanize "github.com/dustin/go-humanize"
	"os"
	//"github.com/jlaffaye/ftp"
	ftp "code.google.com/p/ftp4go"
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

type countWriter struct {
	written int64
	w       io.Writer
}

func (c *countWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.written += int64(n)
	return n, err
}

func StressReads(con *ftp.FTP, files []string, iters int) (int64, error) {
	var nread int64
	for i := 0; i < iters; i++ {
		fi := files[rand.Intn(len(files))]
		cw := &countWriter{w: ioutil.Discard}

		err := con.GetBytes(ftp.RETR_FTP_CMD, cw, ftp.BLOCK_SIZE, fi)
		if err != nil {
			fmt.Println("1 err: ", err)
			return nread, err
		}

		nread += cw.written
	}
	return nread, nil
}

func GetFileList(file string) ([]string, error) {
	fi, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	var out []string
	scan := bufio.NewScanner(fi)
	for scan.Scan() {
		out = append(out, scan.Text())
	}
	return out, nil
}

func main() {
	runtime.GOMAXPROCS(4)

	host := flag.String("host", "localhost", "hostname of ftp server")
	port := flag.Int("port", 21, "port of ftp server")
	threads := flag.Int("threads", 1, "number of concurrent threads to run")
	user := flag.String("user", "test-user", "username")
	pass := flag.String("pass", "password", "password")
	iters := flag.Int("iter", 1, "number of iterations per thread")
	rfile := flag.String("file", "", "number of iterations per thread")
	flist := flag.String("file-list", "", "file containing list of files to read")

	flag.Parse()

	var files []string
	if *flist != "" {
		fs, err := GetFileList(*flist)
		if err != nil {
			fmt.Println(err)
			return
		}
		files = fs
	}

	if files == nil {
		if *rfile == "" {
			fmt.Println("Please specify a file to read with `-file=X` or `-file-list=Y`")
			return
		}
		files = []string{*rfile}
	}

	// temp

	donech := make(chan int64)
	starttime := time.Now()
	for i := 0; i < *threads; i++ {
		go func() {
			ftpC := ftp.NewFTP(0)
			resp, err := ftpC.Connect(*host, *port, "")
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("connect reponse: ", resp.Message)

			resp, err = ftpC.Login(*user, *pass, "")
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("login response: ", resp.Message)
			n, err := StressReads(ftpC, files, *iters)
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
