package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/darling-kefan/xj/nstat"
	"github.com/darling-kefan/xj/nstat/protocol"
)

type Grades struct {
	CurrentScore float64 `json:"current_score"`
	FinalScore   float64 `json:"final_score"`
}

type Enrollment struct {
	ID       int    `json:"id"`
	CourseId int    `json:"course_id"`
	Type     string `json:"type"`
	Grades   Grades `json:"grades"`
}

type User struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	ShortName   string       `json:"short_name"`
	LoginId     string       `json:"login_id"`
	Enrollments []Enrollment `json:"enrollments"`
}

func consumer(statChan chan *protocol.StatFactor, stopMainCh chan struct{}, stopConsumerCh chan struct{}) {
	log.Println("Consumer start...")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	factors := make([]*protocol.StatFactor, 10)
loop:
	for {
		select {
		case <-ticker.C:
			// 一次性最多处理10条消息
			count := 0
		inner:
			for ; count < 10; count++ {
				select {
				case data := <-statChan:
					factors[count] = data
				default:
					break inner
					// 如果statChan中无数据，则执行default.
				}
			}

			// 请求接口，同步到ssdb
			attempt := 1
			for {
				if err := nstat.CommitFactors(factors[:count]); err != nil {
					log.Printf("Failed to commit factors, error: %s\n", err)
					attempt = attempt + 1
				} else {
					break
				}
				if attempt > 3 {
					close(stopMainCh)
					break loop
				}
			}
		case <-stopConsumerCh:
			i := 0
			for data := range statChan {
				factors[i] = data
				i = i + 1
			}
			// 请求接口，同步到ssdb
			attempt := 1
			for {
				if err := nstat.CommitFactors(factors[:i]); err != nil {
					log.Printf("Failed to commit factors, error: %s\n", err)
					attempt = attempt + 1
				} else {
					break
				}
				if attempt > 3 {
					close(stopMainCh)
					break loop
				}
			}
			break loop
		}
	}

	log.Println("Consumer quit...")
}

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Main start...")

	q := url.Values{}
	q.Add("enrollment_state[]", "active")
	q.Add("enrollment_state[]", "invited")
	q.Add("enrollment_type[]", "student")
	q.Add("enrollment_type[]", "student_view")
	q.Add("include[]", "avatar_url")
	q.Add("include[]", "group_ids")
	q.Add("include[]", "enrollments")
	q.Set("per_page", "5")

	// 统计因子通道
	statChan := make(chan *protocol.StatFactor, 10)
	// 停止通道
	stopMainChan := make(chan struct{}, 1)
	stopConsumerChan := make(chan struct{}, 1)

	// 启动协程，将统计因子发送给api
	go consumer(statChan, stopMainChan, stopConsumerChan)

	// Create Http Client
	client := &http.Client{}

	var courses []int = []int{55}
loop:
	for _, courseId := range courses {
		u, err := url.Parse(fmt.Sprintf("https://canvas.ndmooc.com/api/v1/courses/%d/users", courseId))
		if err != nil {
			log.Println(err)
			break loop
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
			var users []User
			if err := json.Unmarshal(body, &users); err != nil {
				log.Println(err)
				break loop
			}
			log.Printf("%#v\n", users)

			for _, user := range users {
				userScore := user.Enrollments[0].Grades.CurrentScore
				statFactor := &protocol.StatFactor{
					Stype:  "23",
					Oid:    "0",
					Sid:    strconv.Itoa(courseId),
					Subkey: strconv.Itoa(user.ID),
					Value:  userScore,
				}

				// 写入statChan, 并监听stopMainChan
				select {
				case statChan <- statFactor:
				case <-stopMainChan:
					break loop
				}
			}

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

	// 通知消费者协程结束程序
	close(stopConsumerChan)
	// 关闭通道，结束进程
	close(statChan)

	log.Println("Main quit...")
}
