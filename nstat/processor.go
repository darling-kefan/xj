package nstat

// 日志处理器协程(Processor)：
//
// <- processor.inbound取出消息，解析成相应的消息类型；生成消息统计因子，并将其发往processor.outbound。

import (
	"log"
	"strconv"
	"sync"

	"github.com/darling-kefan/xj/nstat/protocol"
)

type processor struct {
	inbound  chan *protocol.LogMsg
	outbound chan *protocol.StatData
}

func newProcessor() *processor {
	return &processor{
		inbound:  make(chan *protocol.LogMsg, 10),
		outbound: make(chan *protocol.StatData, 10),
	}
}

func (p *processor) handle(logMsg *protocol.LogMsg) (statData *protocol.StatData, err error) {
	statData = &protocol.StatData{
		LogHeader: logMsg.LogHeader,
		Factors:   make([]*protocol.StatFactor, 0),
	}
	switch logMsg.Mtype {
	case protocol.LOG_ORG_USER_BIND:
		if logMsg.Act == "add" {
			// 生成用户总数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_USER,
				Oid:   logMsg.Oid,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}

			// 生成新增用户总数统计因子
			factor2 := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_NEW_USER,
				Oid:   logMsg.Oid,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor, factor2)
		} else if logMsg.Act == "del" {
			// 生成用户总数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_USER,
				Oid:   logMsg.Oid,
				Value: -1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor)
		}
	case protocol.LOG_USER_LOGIN:
		// 生成用户登录数统计因子
		factor := &protocol.StatFactor{
			Stype:  protocol.STAT_COUNT_LOGIN,
			Subkey: logMsg.Uid,
			Value:  1,
			Date:   logMsg.CreatedAt.Format("2006-01-02"),
		}
		// 生成用户位置数统计因子
		// 根据ip地址地区名称,最后根据地区名称获得did
		did := "0"
		if logMsg.IP != "" {
			did, err = ip2did(logMsg.IP)
			if err != nil {
				return
			}
		}
		factor2 := &protocol.StatFactor{
			Stype:  protocol.STAT_COUNT_LOGIN_DISTRICT,
			Sid:    did,
			Subkey: logMsg.Uid,
			Value:  1,
			Date:   logMsg.CreatedAt.Format("2006-01-02"),
		}
		// 登录排行统计因子(老师,学生)
		if cache.IsTeacher(logMsg.Uid) {
			factor3 := &protocol.StatFactor{
				Stype:  protocol.STAT_COUNT_LOGIN_TEACHER,
				Sid:    "1",
				Subkey: logMsg.Uid,
				Value:  1,
				Date:   logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor, factor2, factor3)
		} else {
			factor3 := &protocol.StatFactor{
				Stype:  protocol.STAT_COUNT_LOGIN_STUDENT,
				Sid:    "2",
				Subkey: logMsg.Uid,
				Value:  1,
				Date:   logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor, factor2, factor3)
		}
	case protocol.LOG_COURSE:
		if logMsg.Act == "add" {
			// 生成课程数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Sid,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}

			// 生成课程评分统计因子
			factor2 := &protocol.StatFactor{
				Stype: protocol.STAT_RATING_COURSE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Sid,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}

			statData.Factors = append(statData.Factors, factor, factor2)
		} else if logMsg.Act == "del" {
			// 生成课程数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Sid,
				Value: -1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}

			// 生成课程评分统计因子
			factor2 := &protocol.StatFactor{
				Stype: protocol.STAT_RATING_COURSE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Sid,
				Value: -1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}

			statData.Factors = append(statData.Factors, factor, factor2)
		}

	case protocol.LOG_COURSE_USER_BIND:
		if logMsg.Act == "add" {
			// 生成课程用户数统计因子
			factor := &protocol.StatFactor{
				Stype:  protocol.STAT_COUNT_COURSE_USER,
				Oid:    logMsg.Oid,
				Sid:    logMsg.Sid,
				Subkey: logMsg.Subkey,
				Value:  1,
				Date:   logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor)
		} else if logMsg.Act == "del" {
			// 生成课程用户数统计因子
			factor := &protocol.StatFactor{
				Stype:  protocol.STAT_COUNT_COURSE_USER,
				Oid:    logMsg.Oid,
				Sid:    logMsg.Sid,
				Subkey: logMsg.Subkey,
				Value:  -1,
				Date:   logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor)
		}
	case protocol.LOG_COURSEWARE:
		var filesize int
		filesize, err = strconv.Atoi(logMsg.Filesize)
		if logMsg.Act == "add" {
			// 生成课件总数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSEWARE,
				Oid:   logMsg.Oid,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件大小统计因子
			factor2 := &protocol.StatFactor{
				Stype: protocol.STAT_SIZE_COURSEWARE,
				Oid:   logMsg.Oid,
				Value: filesize,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件类型数量统计因子
			factor3 := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSEWARE_TYPE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Filetype,
				Value: 1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件类型大小统计因子
			factor4 := &protocol.StatFactor{
				Stype: protocol.STAT_SIZE_COURSEWARE_TYPE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Filetype,
				Value: filesize,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor, factor2, factor3, factor4)
		} else if logMsg.Act == "del" {
			// 生成课件总数统计因子
			factor := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSEWARE,
				Oid:   logMsg.Oid,
				Value: -1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件大小统计因子
			factor2 := &protocol.StatFactor{
				Stype: protocol.STAT_SIZE_COURSEWARE,
				Oid:   logMsg.Oid,
				Value: -filesize,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件类型数量统计因子
			factor3 := &protocol.StatFactor{
				Stype: protocol.STAT_COUNT_COURSEWARE_TYPE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Filetype,
				Value: -1,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			// 生成课件类型大小统计因子
			factor4 := &protocol.StatFactor{
				Stype: protocol.STAT_SIZE_COURSEWARE_TYPE,
				Oid:   logMsg.Oid,
				Sid:   logMsg.Filetype,
				Value: -filesize,
				Date:  logMsg.CreatedAt.Format("2006-01-02"),
			}
			statData.Factors = append(statData.Factors, factor, factor2, factor3, factor4)
		}
	}
	return
}

func (p *processor) run(wg *sync.WaitGroup) {
	log.Println("Processor start...")
	defer wg.Done()
	for {
		select {
		case logMsg := <-p.inbound:
			statData, err := p.handle(logMsg)
			if err != nil {
				log.Println(err)
				// 跳出select语句
				break
			}
			if statData.Topic != "" && statData.Partition != "" && statData.Offset != "" {
				p.outbound <- statData
			}
		case <-stopCh:
			log.Println("Processor quit...")
			return
		}
	}
	log.Println("Processor quit...")
}
