package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
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
	if remote == "" {
		panic("[remote] nexus repository name should not be empty")
	}
	if local == "" {
		panic("[local] maven repository path should not be empty")
	}
	if username == "" {
		panic("[username] of nexus should not be empty")
	}
	if password == "" {
		panic("[password] of nexus user should not be empty")
	}
}

func main() {
	rest := host + "/service/rest/v1/components?repository=" + remote
	asmap := artifactmap(local)
	for key, files := range asmap {
		if len(files) == 1 && strings.HasSuffix(files[0], ".jar") {
			delete(asmap, key)
		} else {
			/**
			req, err := http.NewRequest("POST", rest, nil)
			req.SetBasicAuth(username, password)
			if err != nil {
				panic(err)
			}
			req.Header.Add("accept", "application/json")
			req.Header.Add("Content-Type", "multipart/form-data")
			form := url.Values{"maven2.generate-pom": []string{"false"}}
			for i, f := range files {
				count := i + 1
				if strings.HasSuffix(f, ".pom") {
					form.Add(fmt.Sprintf("maven2.asset%d", count), "@"+f)
					form.Add(fmt.Sprintf("maven2.asset%d.extension", count), "pom")
				} else if strings.HasSuffix(f, ".jar") {
					form.Add(fmt.Sprintf("maven2.asset%d", count), "@"+f+";type=application/java-archive")
					form.Add(fmt.Sprintf("maven2.asset%d.extension", count), "jar")
				}
			}
			req.PostForm = form
			fmt.Println(req.PostForm)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			fmt.Println(resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(body))
			defer resp.Body.Close()
			*/
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			form := url.Values{}
			for i, f := range files {
				file, _ := os.Open(f)
				defer file.Close()
				count := i + 1
				if strings.HasSuffix(f, ".pom") {
					part, _ := writer.CreateFormFile(fmt.Sprintf("maven2.asset%d", count), filepath.Base(file.Name()))
					io.Copy(part, file)
					form.Add(fmt.Sprintf("maven2.asset%d.extension", count), "pom")
				} else if strings.HasSuffix(f, ".jar") {
					part, _ := writer.CreateFormFile(fmt.Sprintf("maven2.asset%d", count), filepath.Base(file.Name()))
					io.Copy(part, file)
					form.Add(fmt.Sprintf("maven2.asset%d.extension", count), "jar")
				}
			}
			writer.Close()
			r, _ := http.NewRequest("POST", rest, body)
			r.Header.Add("Content-Type", writer.FormDataContentType())
			client := &http.Client{}
			resp, err := client.Do(r)
			if err != nil {
				panic(err)
			}
			respbody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(respbody))
			defer resp.Body.Close()
		}
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
