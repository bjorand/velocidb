package core

func Help(topic string) string {
	help := make(map[string]string)
	help["peer"] = `
peer connect <host>:<port>
peer list
peer remove <id>
  `
	if topic != "" && help[topic] != "" {
		return help[topic]
	}

	var full string
	for _, text := range help {
		full += text
	}
	return full
}
