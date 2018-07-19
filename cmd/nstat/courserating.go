package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/nstat"
	"github.com/darling-kefan/xj/nstat/protocol"
	_ "github.com/go-sql-driver/mysql"
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

type CanvasErrInfo struct {
	Message string `json:"message"`
}

type CanvasErrs struct {
	Errors []CanvasErrInfo `json:"errors"`
}

type User struct {
	ID          int          `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"`
	ShortName   string       `json:"short_name,omitempty"`
	LoginId     string       `json:"login_id,omitempty"`
	Enrollments []Enrollment `json:"enrollments,omitempty"`
}

// 计算评分等级
func gradingStandards(val float64) string {
	var grade string
	switch {
	case val >= 94:
		grade = "A"
	case val >= 90:
		grade = "A-"
	case val >= 87:
		grade = "B+"
	case val >= 84:
		grade = "B"
	case val >= 80:
		grade = "B-"
	case val >= 77:
		grade = "C+"
	case val >= 74:
		grade = "C"
	case val >= 70:
		grade = "C-"
	case val >= 67:
		grade = "D+"
	case val >= 64:
		grade = "D"
	case val >= 61:
		grade = "D-"
	default:
		grade = "F"
	}
	return grade
}

func getAllCourses() []string {
	apiConf := config.Config.MySQL["api"]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		apiConf.Username,
		apiConf.Password,
		apiConf.Host,
		apiConf.Port,
		apiConf.Dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	sql := "SELECT `course_id` FROM `courses` WHERE `deleted_at` IS NULL"
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var courseId string
	courses := make([]string, 0)
	for rows.Next() {
		err := rows.Scan(&courseId)
		if err != nil {
			log.Fatal(err)
		}
		courses = append(courses, courseId)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return courses
}

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Main start...")

	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

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

	// TODO 获取所有课程
	var courses []string = getAllCourses()
	log.Printf("All courses: %v\n", courses)

loop:
	for _, courseId := range courses {
		// 每个课程所有评分等级对应的人数
		gradeUsers := make(map[string]int)

		u, err := url.Parse(fmt.Sprintf("%s/api/v1/courses/%s/users", config.Config.Common.CanvasHost, courseId))
		if err != nil {
			log.Println(err)
			break loop
		}
		u.RawQuery = q.Encode()

		urlStr := u.String()
		for {
			//log.Println(urlStr)
			r, _ := http.NewRequest("GET", urlStr, nil)
			r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.Config.Common.CanvasToken))
			resp, _ := client.Do(r)
			defer resp.Body.Close()

			body, _ := ioutil.ReadAll(resp.Body)
			var canvasErrs CanvasErrs
			var users []User
			if err := json.Unmarshal(body, &canvasErrs); err == nil {
				log.Printf("Failed to request %s, resp: %s\n", urlStr, string(body))
				break
			} else {
				if err := json.Unmarshal(body, &users); err != nil {
					log.Println(err)
					break loop
				}
			}
			log.Printf("Courser users: %#v\n", users)

			for _, user := range users {
				userScore := user.Enrollments[0].Grades.CurrentScore
				if userScore > 0 {
					grade := gradingStandards(userScore)
					if _, ok := gradeUsers[grade]; ok {
						gradeUsers[grade] = gradeUsers[grade] + 1
					} else {
						gradeUsers[grade] = 1
					}
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

		log.Printf("Grading standards: %v", gradeUsers)

		// 根据课程courseId获取机构

		factors := make([]*protocol.StatFactor, 0)
		for gradeStandard, userCount := range gradeUsers {
			statFactor := &protocol.StatFactor{
				Stype:  "23",
				Oid:    "0",
				Sid:    courseId,
				Subkey: gradeStandard,
				Value:  float64(userCount),
			}
			factors = append(factors, statFactor)
		}

		// 请求接口，同步到ssdb
		attempt := 1
		for {
			if err := nstat.CommitFactors(factors); err != nil {
				log.Printf("Failed to commit factors, error: %s\n", err)
				attempt = attempt + 1
			} else {
				break
			}
			if attempt > 3 {
				break loop
			}
		}
	}

	log.Println("Main quit...")
}
