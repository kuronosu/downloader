package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type partial struct {
	index int
	data  []byte
}

func getFileName(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func save(filename string, data []byte) {
	fmt.Println("Saving file ", filename)
	startTime := time.Now()
	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		fmt.Println("Error")
	}
	fmt.Printf("Saved %d bytes in %s\n", len(data), time.Since(startTime))
}

func getSize(url string) int {
	resp, err := http.Head(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		os.Exit(1)
	}
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	return size
}

func downloadPartial(url string, offset, size int) []byte {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", offset, size))
	var client http.Client
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func sortParts(parts map[int][]byte) []byte {
	startTime := time.Now()
	keys := make([]int, 0)
	for k := range parts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	data := parts[0]
	for _, k := range keys {
		if k == 0 {
			continue
		}
		val := parts[k]
		data = append(data, val...)
	}
	fmt.Printf("Joined parts in %s\n", time.Since(startTime))
	return data
}

func download(url string, parts int) []byte {
	startTime := time.Now()
	m := make(map[int][]byte)
	size := getSize(url)
	partSize := size / parts
	fmt.Println("Size: ", size, "\tPart size: ", partSize)
	ch := make(chan partial)
	for i := 0; i < parts; i++ {
		go func(_i int, _ch chan partial) {
			offset := 0
			if _i != 0 {
				offset = (partSize * _i) + 1
			}
			_size := offset + partSize
			if _i != 0 {
				_size -= 1
			}
			if _i == parts-1 {
				_size = size
			}
			// fmt.Println(offset, "-", _size)
			_ch <- partial{index: _i, data: downloadPartial(url, offset, _size)}
		}(i, ch)
	}
	i := 0
	for {
		valor := <-ch
		m[valor.index] = valor.data
		i += 1
		if i == parts {
			break
		}
	}
	fmt.Printf("Downloaded %d bytes in %s\n", size, time.Since(startTime))
	return sortParts(m)
}

func getParameters() (string, int, string) {
	fmt.Println("Start")
	if len(os.Args) < 3 {
		fmt.Println("Missing url or parts")
		os.Exit(1)
	}
	url := os.Args[1]
	parts, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if parts <= 0 {
		fmt.Println("Parts must be positive")
		os.Exit(1)
	}
	filename := ""
	if len(os.Args) >= 4 {
		filename = os.Args[3]
	} else {
		filename = getFileName(url)
	}
	return url, parts, filename
}

func main() {
	url, parts, filename := getParameters()
	data := download(url, parts)
	save(filename, data)
}
