package request

// CasbinInfo Casbin 策略信息结构
type CasbinInfo struct {
	Path   string `json:"path"`   // 路径
	Method string `json:"method"` // 方法
}

// CasbinInReceive Casbin 策略的入参结构
type CasbinInReceive struct {
	AuthorityId uint         `json:"authorityId"` // 权限id
	CasbinInfos []CasbinInfo `json:"casbinInfos"`
}

func DefaultCasbin() []CasbinInfo {
	return []CasbinInfo{
		{Path: "/menu/getMenu", Method: "POST"},
		{Path: "/jwt/jsonInBlacklist", Method: "POST"},
		{Path: "/base/login", Method: "POST"},
		{Path: "/user/changePassword", Method: "POST"},
		{Path: "/user/setUserAuthority", Method: "POST"},
		{Path: "/user/getUserInfo", Method: "GET"},
		{Path: "/user/setSelfInfo", Method: "PUT"},
		{Path: "/fileUploadAndDownload/upload", Method: "POST"},
		{Path: "/sysDictionary/findSysDictionary", Method: "GET"},
	}
}
