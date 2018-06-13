package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	q := url.Values{}
	q.Add("enrollment_state[]", "active")
	q.Add("enrollment_state[]", "invited")
	q.Add("enrollment_type[]", "student")
	q.Add("enrollment_type[]", "student_view")
	q.Add("include[]", "avatar_url")
	q.Add("include[]", "group_ids")
	q.Add("include[]", "enrollments")
	q.Set("per_page", "5")

	// Create Http Client
	client := &http.Client{}

	var courses []int = []int{55}
	for _, courseId := range courses {
		u, err := url.Parse(fmt.Sprintf("https://canvas.ndmooc.com/api/v1/courses/%d/users", courseId))
		if err != nil {
			log.Fatal(err)
		}
		u.RawQuery = q.Encode()

		urlStr := u.String()
		for {
			//log.Println(urlStr)
			r, _ := http.NewRequest("GET", urlStr, nil)
			r.Header.Add("Authorization", "Bearer uUDNCm1ViEun51G7qJr7MdFplegLHA8MSirAbfntC9YcAz0YnhnsShxPo9URNT1u")
			resp, _ := client.Do(r)
			defer resp.Body.Close()

			body, _ := ioutil.ReadAll(resp.Body)
			log.Println(string(body))

			// Canvas分页读取列表
			var hasNext bool
			links := strings.Split(resp.Header["Link"][0], ",")
			for _, link := range links {
				linkParts := strings.Split(link, ";")
				if strings.Contains(linkParts[1], "next") {
					hasNext = true
					urlStr = strings.Trim(linkParts[0], "<>")
					break
				}
			}
			if !hasNext {
				break
			}
		}

	}
}
