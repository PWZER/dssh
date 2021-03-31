package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func JsonFormatPrint(content []byte) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(content, &obj); err != nil {
		return err
	}
	if new_content, err := json.MarshalIndent(obj, "", "    "); err != nil {
		return err
	} else {
		fmt.Println(string(new_content))
	}
	return nil
}

func JsonPrint(contents []string) error {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		var content []byte
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			content = append(content, scanner.Bytes()...)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return JsonFormatPrint(content)
	}

	for _, content := range contents {
		if err := JsonFormatPrint([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}
