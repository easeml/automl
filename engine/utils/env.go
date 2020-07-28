package utils

import "os"

func GetEnvVariableOrDefault(envarname string,defaultval string) string {
	envvar := os.Getenv(envarname)
	println("ENV ",envvar)
	if envvar == "" {
		return defaultval
	}
	return envvar
}

