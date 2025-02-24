package manager

type ProxyConfig struct {
	Proxy struct {
		Enable     bool              `json:"enable"`
		SameForAll bool              `json:"same_for_all"`
		Servers    map[string]string `json:"servers"`
		Bypass     string            `json:"bypass"`
	} `json:"proxy"`
	PAC struct {
		Enable bool   `json:"enable"`
		URL    string `json:"url"`
	} `json:"pac"`
}
