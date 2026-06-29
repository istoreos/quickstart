package service

func ScopeMessage(code int, scope, message, lang string) (string, string) {
	// TODO
	if lang == "" {
		lang = "zh_CN"
	}
	return message, ""
}
