package nstat

import (
	"log"
	"strconv"
)

// 根据ip地址查找地区id
func ip2did(ip string) (did string, err error) {
	var arr []string
	arr, err = cityipdb.Find(ip)
	if err != nil {
		return
	}
	if len(arr) >= 3 {
		if arr[2] != "" {
			cityname := arr[2]
			did = strconv.Itoa(districtDb.City[cityname])
		} else if arr[1] != "" {
			provincename := arr[1]
			did = strconv.Itoa(districtDb.Province[provincename])
		} else if arr[0] != "" {
			countryname := arr[0]
			did = strconv.Itoa(districtDb.Country[countryname])
		}
	}
	log.Printf("ipip: %v did: %v err: %v\n", arr, did, err)
	return
}
