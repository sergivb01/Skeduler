package database

import (
	"fmt"
	"strings"
)

const envSplit = "_#&#_"

func envToString(env map[string]interface{}) string {
	var sb strings.Builder
	for k, v := range env {
		sb.WriteString(k)
		sb.WriteRune('=')
		sb.WriteString(fmt.Sprintf("%v", v))
		sb.WriteString(envSplit)
	}
	return strings.TrimSuffix(sb.String(), envSplit)
}

func stringToEnv(val string) map[string]interface{} {
	env := make(map[string]interface{})
	for _, kv := range strings.Split(val, envSplit) {
		parts := strings.SplitN(kv, "=", 2)
		env[parts[0]] = parts[1]
	}

	return env
}
