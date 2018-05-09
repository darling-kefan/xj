package ndscloud

// 设备详情
type DeviceInfo struct {
	ClientId   string     `json:"client_id"`
	Scope      string     `json:"scope"`
	Permission Permission `json:"permission,omitempty"`
	Config     Config     `json:"config,omitempty"`
}

type Config struct {
	Device Device `json:"device"`
}

type Device struct {
	DeviceType     int    `json:"device_type"`
	ClassroomId    string `json:"classroom_id"`
	ClassroomTitle string `json:"classroom_title"`
}
