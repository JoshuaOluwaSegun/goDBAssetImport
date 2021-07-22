package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/color"
	apiLib "github.com/hornbill/goApiLib"
)

func initXMLMC() {

	hornbillImport = apiLib.NewXmlmcInstance(SQLImportConf.InstanceID)
	hornbillImport.SetAPIKey(SQLImportConf.APIKey)
	hornbillImport.SetTimeout(60)
	hornbillImport.SetJSONResponse(true)

	pageSize = 0

	if pageSize == 0 {
		pageSize = 100
	}
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(k string, v string, u map[string]interface{}, buffer *bytes.Buffer) string {
	debugLog(buffer, "getFieldValue:", k, ":", v)
	fieldMap := v
	if fieldMap == "[HBAssetType]" {
		debugLog(buffer, "Returning AssetType:", StrAssetType)
		return StrAssetType
	}

	t := template.New(fieldMap).Funcs(TemplateFilters)
	tmpl, _ := t.Parse(fieldMap)
	buf := bytes.NewBufferString("")
	tmpl.Execute(buf, u)
	value := buf.String()
	debugLog(buffer, "value:", value)

	if value == "%!s(<nil>)" || value == "<no value>" {
		value = ""
		}
	fieldMap = value
	if fieldMap != "" {
			if strings.Contains(strings.ToLower(k), "date") {
			fieldMap = checkDateString(fieldMap)
			}
			}
	debugLog(buffer, "returning:", fieldMap)
	return fieldMap
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool, outputToEsp bool) {
	//-- Current working dir
	cwd, _ := os.Getwd()

	//-- Log Folder
	logPath := cwd + "/log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Log File
	logFileName := logPath + "/Asset_Import_" + startTime.Format("20060102150405") + "_" + strconv.Itoa(logFilePart) + ".log"
	if maxLogFileSize > 0 {
		//Check log file size
		fileLoad, e := os.Stat(logFileName)
		if e != nil {
			//File does not exist - do nothing!
		} else {
			fileSize := fileLoad.Size()
			if fileSize > maxLogFileSize {
				logFilePart++
				logFileName = logPath + "/Asset_Import_" + startTime.Format("20060102150405") + "_" + strconv.Itoa(logFilePart) + ".log"
			}
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// don't forget to close it
	defer f.Close()
	var (
		errorLogPrefix string
		espLogType     string
	)
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		espLogType = "debug"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		espLogType = "notice"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		espLogType = "notice"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		espLogType = "error"
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	case 5:
		errorLogPrefix = "[WARNING] "
		espLogType = "warn"
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	}
	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	if outputToEsp {
		espLogger(s, espLogType)
	}
	mutex.Lock()
	// assign it to the standard logger
	log.SetOutput(f)
	log.Println(errorLogPrefix + s)
	mutex.Unlock()
}

func loggerGen(t int, s string) string {

	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = ""
	case 4:
		errorLogPrefix = "[ERROR] "
	case 5:
		errorLogPrefix = "[WARNING] "
	}
	return errorLogPrefix + s + "\n\r"
}
func loggerWriteBuffer(s string) {
	if s != "" {
		logLines := strings.Split(s, "\n\r")
		for _, line := range logLines {
			if line != "" {
				logger(0, line, false, false)
			}
		}
	}
}

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	strNewDate := re.FindString(strDate)
	return strNewDate
}

func debugLog(buffer *bytes.Buffer, debugStrings ...string) {
	if configDebug {
		if buffer == nil {
			logger(1, strings.Join(debugStrings, " "), false, false)
		} else {
			buffer.WriteString(loggerGen(1, strings.Join(debugStrings, " ")))
		}
	}
}

func iToS(interfaceVal interface{}) (strVal string) {
	if interfaceVal == nil {
		return
	}

	switch v := interfaceVal.(type) {
	case []uint8:
		strVal = string(v)
	default:
		strVal = fmt.Sprintf("%v", interfaceVal)
	}
	return
}

func Hash(arr []map[string]interface{}) string {
	arrBytes := []byte{}
	for _, item := range arr {
		jsonBytes, _ := json.Marshal(item)
		arrBytes = append(arrBytes, jsonBytes...)
	}
	has := md5.Sum(arrBytes)

	md5str := fmt.Sprintf("%x", has)
	return md5str
}

// espLogger -- Log to ESP
func espLogger(message string, severity string) {
	if configDryRun {
		message = "[DRYRUN] " + message
	}
	hornbillImport.SetParam("fileName", appName)
	hornbillImport.SetParam("group", "general")
	hornbillImport.SetParam("severity", severity)
	hornbillImport.SetParam("message", message)
	hornbillImport.Invoke("system", "logMessage")
}
