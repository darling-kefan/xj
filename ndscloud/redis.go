package ndscloud

var (
	// 存储最新场景id(kv)
	// fmt.Sprintf(this, unitId)
	sceneIdKeyFormat string = "nc:unit:scene:id:%s"
	// 存储单元场景信息(kv)
	// fmt.Sprintf(this, unitId, sceneId)
	sceneKeyFormat string = "nc:unit:scene:%s:%d"

	// 存储模块状态指令(hash: instr_mod -> content)
	// fmt.Sprintf(this, unitId, sceneId)
	modInsKeyFormat string = "nc:ins:mod:%s:%d"
	// 模块状态指令历史前缀
	// fmt.Sprintf(this, unitId, sceneId, curmod)
	modInsHistoryKeyFormat string = "nc:ins:mod:his:%s:%d:%s"

	// 群聊(文字聊天)(list)
	// fmt.Sprintf(this, unitId, sceneId)
	chatKeyFormat string = "nc:chat:his:%s:%d"
)
