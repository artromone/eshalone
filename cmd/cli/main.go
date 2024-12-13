package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const serverURL = "http://localhost:8080"

func sendRequest(endpoint, employeeID string) error {
	url := fmt.Sprintf("%s/%s?employee_id=%s", serverURL, endpoint, employeeID)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка: %s", string(body))
	}

	var result map[string]string
	json.Unmarshal(body, &result)
	fmt.Println(result["status"])

	return nil
}

func formatTimerEntries(entries []map[string]interface{}) string {
	var buffer bytes.Buffer
	for i, entry := range entries {
		buffer.WriteString(fmt.Sprintf("Запись %d:\n", i+1))
		buffer.WriteString(fmt.Sprintf("  Начало: %s\n", entry["start_time"]))
		
		if running, ok := entry["is_running"].(bool); ok && !running {
			buffer.WriteString(fmt.Sprintf("  Конец: %s\n", entry["end_time"]))
			buffer.WriteString(fmt.Sprintf("  Продолжительность: %s\n", entry["duration"]))
		} else {
			buffer.WriteString("  Таймер активен\n")
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func main() {
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	startEmployeeID := startCmd.String("id", "", "ID сотрудника")
	stopEmployeeID := stopCmd.String("id", "", "ID сотрудника")
	infoEmployeeID := infoCmd.String("id", "", "ID сотрудника")

	if len(os.Args) < 2 {
		fmt.Println("Используйте: timer-cli [start|stop|info] -id=EMPLOYEE_ID")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		startCmd.Parse(os.Args[2:])
		if *startEmployeeID == "" {
			fmt.Println("Требуется ID сотрудника")
			os.Exit(1)
		}
		err := sendRequest("start", *startEmployeeID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "stop":
		stopCmd.Parse(os.Args[2:])
		if *stopEmployeeID == "" {
			fmt.Println("Требуется ID сотрудника")
			os.Exit(1)
		}
		err := sendRequest("stop", *stopEmployeeID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "info":
		infoCmd.Parse(os.Args[2:])
		if *infoEmployeeID == "" {
			fmt.Println("Требуется ID сотрудника")
			os.Exit(1)
		}
		url := fmt.Sprintf("%s/info?employee_id=%s", serverURL, *infoEmployeeID)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var timerInfo map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&timerInfo)
		
		entries, _ := timerInfo["entries"].([]interface{})
		formattedEntries := make([]map[string]interface{}, len(entries))
		for i, entry := range entries {
			formattedEntries[i] = entry.(map[string]interface{})
		}

		fmt.Println(formatTimerEntries(formattedEntries))

	default:
		fmt.Println("Используйте: timer-cli [start|stop|info] -id=EMPLOYEE_ID")
		os.Exit(1)
	}
}
