package main

import (
	"crypto/md5"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

var (
	c, c_err = redis.Dial("tcp", ":6379")
)

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	reply, err := redis.String(c.Do("GET", key))
	if reply == "" || err != nil {
		fmt.Fprintf(w, "Error: %s not found.", html.EscapeString(r.URL.Path))
	} else {
		count := key + ":count"
		c.Do("EXPIRE", key, 43200)
		c.Do("INCR", count)
		c.Do("EXPIRE", count, 43200)
		http.Redirect(w, r, "http://"+reply, 301)
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	const lenPath = len("/add/")
	value := r.URL.Path[lenPath:]
	fmt.Println(value)
	h := md5.New()
	io.WriteString(h, r.URL.Path)
	s := fmt.Sprintf("%x", h.Sum(nil))
	s = s[:4]
	c.Do("SETEX", s, 43200, value)
	fmt.Fprintf(w, "Added: %s as %s.", html.EscapeString(value), html.EscapeString(s))
}

type ShortenedURLList struct {
	ShortenedURLs []*ShortenedURL
}

type ShortenedURL struct {
	Short string
	Long  string
	Count int
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	t := template.New("shortened")
	t, err := t.ParseFiles("tmpl/base.html")
	if err != nil {
		fmt.Println(err)
	}

	reply, err := redis.Values(c.Do("KEYS", "*"))
	if err != nil {
		fmt.Fprintf(w, "Internal error: %s", err)
	}

	a := ShortenedURLList{}

	for _, v := range reply {
		v_ := fmt.Sprintf("%s", v)
		if strings.Contains(v_, ":") {
			continue
		}
		url, _ := redis.String(c.Do("GET", v))
		count, _ := redis.Int(c.Do("GET", fmt.Sprintf("%s:count", v)))
		a.ShortenedURLs = append(a.ShortenedURLs, &ShortenedURL{Short: v_, Long: url, Count: count})
	}

	err = t.Execute(w, a)
	if err != nil {
		fmt.Print(err)
	}
}

func main() {
	defer c.Close()
	if c_err != nil {
		log.Fatal(c_err)
	}

	_, err := c.Do("SELECT", 1)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/add/", addHandler)
	http.HandleFunc("/list/", listHandler)
	http.HandleFunc("/", redirectHandler)
	http.ListenAndServe(":8080", nil)
}
