package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

func ParseCmd(s string) []string {
	s = strings.TrimSpace(s)
	return strings.Split(s, " ")
}

func ValidateCmdAndGetKeys(cmd []string) ([]string, error) {
	if len(cmd) == 0 {
		return nil, errors.New("empty command")
	}

	cmd[0] = strings.ToUpper(cmd[0])
	var keys []string
	switch cmd[0] {
	case "SET":
		if len(cmd) != 3 {
			return nil, errors.New("SET command format is: SET key value")
		}
		keys = append(keys, cmd[1])
	case "GET":
		if len(cmd) != 2 {
			return nil, errors.New("GET command format is: GET key")
		}
		keys = append(keys, cmd[1])
	case "SADD":
		if len(cmd) < 3 {
			return nil, errors.New("SADD command format is: SADD key value1 [value2...]")
		}
		keys = append(keys, cmd[1])
	case "SREM":
		if len(cmd) < 3 {
			return nil, errors.New("SREM command format is: SREM key value1 [value2...]")
		}
		keys = append(keys, cmd[1])
	case "SMEMBERS":
		if len(cmd) != 2 {
			return nil, errors.New("SMEMBERS command format is: SMEMBERS key")
		}
		keys = append(keys, cmd[1])
	case "SINTER":
		if len(cmd) < 2 {
			return nil, errors.New("SINTER command format is: SINTER key1 [key2] ...")
		}
		keys = append(keys, cmd[1:]...)
	case "KEYS":
		if len(cmd) != 1 {
			return nil, errors.New("KEYS command format is: KEYS")
		}
	case "DEL":
		if len(cmd) != 2 {
			return nil, errors.New("DEL command format is: DEL key")
		}
		keys = append(keys, cmd[1])
	case "EXPIRE":
		if len(cmd) != 3 {
			return nil, errors.New("EXPIRE command format is: EXPIRE key seconds")
		}
		keys = append(keys, cmd[1])
	case "TTL":
		if len(cmd) != 2 {
			return nil, errors.New("TTL command format is: TTL key")
		}
		keys = append(keys, cmd[1])
	case "SAVE":
		if len(cmd) != 1 {
			return nil, errors.New("SAVE command format is: SAVE")
		}
	case "RESTORE":
		if len(cmd) != 1 {
			return nil, errors.New("RESTORE command format is: RESTORE")
		}
	default:
		return nil, errors.New("Unknown command")
	}
	return keys, nil
}

func Str2TimeDuration(s string) (time.Duration, error) {
	num, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return time.Duration(num) * time.Second, nil
}
