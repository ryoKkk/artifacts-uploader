package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	host,
	remote,
	local,
	username,
	password string
)

func init() {
	flag.StringVar(&host, "host", "http://localhost:8081", "nexus3 host")
	flag.StringVar(&remote, "remote", "", "remote nexus repository name")
	fmt.Println(remote)
	flag.StringVar(&local, "local", "", "local maven repository path, generally, $HOME/.m2/repository")
	flag.StringVar(&username, "username", "", "nexus login username")
	flag.StringVar(&password, "password", "", "nexus login user password")
	flag.Parse()
	es := make([]string, 0, 4)
	if remote == "" {
		es = append(es, "[remote] nexus repository name should not be empty")
	}
	if local == "" {
		es = append(es, "[local] maven repository path should not be empty")
	}
	if username == "" {
		es = append(es, "[username] of nexus should not be empty")
	}
	if password == "" {
		es = append(es, "[password] of nexus user should not be empty")
	}
	if len(es) > 0 {
		panic(strings.Join(es, "\n"))
	}
}

func main() {
	rest := host + "/service/rest/v1/components?repository=" + remote
	asmap := artifactmap(local)
	// var wg sync.WaitGroup
	for key, files := range asmap {
		if len(files) == 1 && strings.HasSuffix(files[0], ".jar") {
			delete(asmap, key)
		} else {
			upload(rest, files)
			// wg.Add(1)
		}
	}
	// wg.Wait()
}

func upload(endpoint string, files []string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for i, f := range files {
		file, _ := os.Open(f)
		defer file.Close()
		count := i + 1
		if strings.HasSuffix(f, ".pom") {
			part, _ := writer.CreateFormFile(fmt.Sprintf("maven2.asset%d", count), filepath.Base(file.Name()))
			io.Copy(part, file)
			pom, _ := writer.CreateFormField(fmt.Sprintf("maven2.asset%d.extension", count))
			pom.Write([]byte("pom"))
		} else if strings.HasSuffix(f, ".jar") {
			part, _ := writer.CreateFormFile(fmt.Sprintf("maven2.asset%d", count), filepath.Base(file.Name()))
			io.Copy(part, file)
			jar, _ := writer.CreateFormField(fmt.Sprintf("maven2.asset%d.extension", count))
			jar.Write([]byte("jar"))
		}
	}
	writer.Close()
	r, _ := http.NewRequest("POST", endpoint, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	r.SetBasicAuth(username, password)
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		msg := "failed in sending request: " + err
		return
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		fmt.Println("file(s) uploaded successfully: ", files)
	} else if statusCode >= 400 && statusCode < 400 {
		fmt.Println("server error 4XX: ", resp)
	} else if statusCode == 500 {
		fmt.Println("server error 500: ", resp)
	} else {
		fmt.Println("unknown response: ", resp)
	}
}

func artifactmap(root string) map[string][]string {
	asmap := make(map[string][]string, 100)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && isPomOrJar(path) {
			name := strings.TrimSuffix(path, filepath.Ext(path))
			as := asmap[name]
			if as == nil {
				as = make([]string, 0, 2)
			}
			as = append(as, path)
			asmap[name] = as
		}
		return nil
	})
	// delete jar not having corresponding pom
	for key, files := range asmap {
		if len(files) == 1 && strings.HasSuffix(files[0], ".jar") {
			delete(asmap, key)
		}
	}
	return asmap
}

func isPomOrJar(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".jar" || ext == ".pom"
}
