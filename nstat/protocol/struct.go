package protocol

// 日志消息头
type LogHeader struct {
	Topic string `json:"-"` // 消息类别，如对应Kafka Topic
	ID    string `json:"-"` // 消息ID
}

// -----------------------------------------------------------------
// 源日志消息格式
// -----------------------------------------------------------------

// Mtype用于标识日志类型
type Mtype string

const (
	LOG_ORG_USER_BIND    Mtype = "1"  // 用户和机构绑定/解绑日志
	LOG_USER_LOGIN       Mtype = "11" // 用户登录日志
	LOG_COURSE           Mtype = "21" // 课程日志
	LOG_COURSE_USER_BIND Mtype = "22" // 课程和用户绑定/解绑日志
	LOG_COURSEWARE       Mtype = "41" // 课件日志
)

// 通用日志消息
type Msg struct {
	LogHeader
	Mtype     Mtype  `json:"mtype"`
	Oid       string `json:"oid"`
	Act       string `json:"act"`
	Sid       string `json:"sid,omitempty"`
	Subkey    string `json:"subkey,omitempty"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
}

// 登录日志消息
type LoginMsg struct {
	LogHeader
	Mtype     Mtype  `json:"mtype"`
	Uid       string `json:"uid"`
	Nickname  string `json:"nickname"`
	IP        string `json:"ip"`
	CreatedAt string `json:"created_at"`
}

// 课件日志消息
type CoursewareMsg struct {
	MsgHeader
	Mtype     Mtype  `json:"mtype"`
	Oid       string `json:"oid"`
	Act       string `json:"act"`
	Filetype  string `json:"filetype"`
	Filesize  string `json:"filesize"`
	CreatedAt string `json:"created_at"`
}

// -----------------------------------------------------------------
// 日志统计因子格式
// -----------------------------------------------------------------

// Stype用于标识统计因子
type Stype string

const (
	STAT_COUNT_USER            Stype = "1" // 用户总数
	STAT_COUNT_NEW_USER        Stype = "2" // 新增用户数
	STAT_COUNT_LOGIN           Stype = "3" // 用户登录数
	STAT_COUNT_LOGIN_DISTRICT  Stype = "4" // 每天登录位置数
	STAT_RANKING_LOGIN_TEACHER Stype = "5" // 老师登录排行
	STAT_RANKING_LOGIN_STUDENT Stype = "6" // 学生登录排行

	STAT_COUNT_COURSE        Stype = "21" // 课程总数
	STAT_RANKING_COURSE_USER Stype = "22" // 用户数排行榜
	STAT_AVERAGE_COURSE_USER Stype = "23" // 平均用户数
	STAT_RATING_COURSE       Stype = "24" // 课程评分

	STAT_COUNT_COURSEWARE      Stype = "41" // 课件总数
	STAT_SIZE_COURSEWARE       Stype = "42" // 课件占用空间
	STAT_COUNT_COURSEWARE_TYPE Stype = "43" // 课件类型数量
	STAT_SIZE_COURSEWARE_TYPE  Stype = "44" // 课件类型大小
)

// 日志统计因子，一条日志对应一个StatData
type StatData struct {
	LogHeader
	Factors []*StatFactor
}

type StatFactor struct {
	Stype  Stype  `json:"stype"`
	Oid    string `json:"oid"`
	Sid    string `json:"sid,omitempty"`
	Subkey string `json:"subkey,omitempty"`
	Value  string `json:"value"`
	Date   string `json:"date,omitempty"`
	//Mod    string `json:"mod"`
}
