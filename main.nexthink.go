package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func getAssetsFromNexthink(assetType assetTypesStruct) (map[string]map[string]interface{}, error) {
	//Initialise Asset Map
	var arrAssetMaps []map[string]interface{}
	returnMap := make(map[string]map[string]interface{})
	logger(3, " ", false, false)
	logger(3, "[NEXTHINK] Running Nexthink query for "+assetType.AssetType+" assets. Please wait...", true, true)

	strUrl := key.Server + "/query?"
	if assetType.NexthinkPlatform != "" {
		strUrl += "platform=" + assetType.NexthinkPlatform + "&"
	}
	strUrl += "query=" + url.QueryEscape(assetType.Query)
	strUrl += "&format=json"
	req, err := http.NewRequest("GET", strUrl, nil)

	if err != nil {
		return returnMap, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(key.Username + ":" + key.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", appName+"/"+version)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Do(req)
	if err != nil {
		return returnMap, err
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(io.Discard, resp.Body)
		return returnMap, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return returnMap, errors.New("cant read the body of the response: " + err.Error())
	}
	if err := json.Unmarshal([]byte(body), &arrAssetMaps); err != nil {
		log.Fatal(err)
	}
	for _, v := range arrAssetMaps {
		assetIdentifier := fmt.Sprintf("%s", v[assetType.AssetIdentifier.SourceColumn])
		returnMap[assetIdentifier] = make(map[string]interface{})
		for field, value := range v {
			switch actualVal := value.(type) {
			case []interface{}:
				returnMap[assetIdentifier][field] = actualVal[len(actualVal)-1]
			case float64:
				if field == "system_drive_capacity" || field == "total_ram" {
					returnMap[assetIdentifier][field] = byteCountSI(actualVal)
				} else {
					returnMap[assetIdentifier][field] = actualVal
				}
			default:
				if field == "last_logon_time" {
					t, _ := time.Parse("2006-01-02T15:04:05", iToS(actualVal))
					actualVal = t.Format("2006-01-02 15:04:05")
				}
				returnMap[assetIdentifier][field] = actualVal
			}
		}
	}
	return returnMap, nil
}

func byteCountSI(b float64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%v B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func queryNexthinkSoftwareInventoryRecords(assetID string, assetType assetTypesStruct, buffer *bytes.Buffer) (map[string]map[string]interface{}, string, error) {

	var (
		returnMap = make(map[string]map[string]interface{})
		hash      string
		err       error
	)
	sqlAssetQuery := strings.ReplaceAll(assetType.SoftwareInventory.Query, "{{AssetID}}", assetID)

	//Initialise Asset Map
	var arrSoftwareMaps []map[string]interface{}
	buffer.WriteString(loggerGen(3, "[NEXTHINK] Running Nexthink query for software against "+assetID+" asset. Please wait..."))

	strUrl := key.Server + "/query?"
	strUrl += "query=" + url.QueryEscape(sqlAssetQuery)
	strUrl += "&format=json"

	req, err := http.NewRequest("GET", strUrl, nil)

	if err != nil {
		return returnMap, hash, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(key.Username + ":" + key.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", appName+"/"+version)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Do(req)
	if err != nil {
		return returnMap, hash, err
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(io.Discard, resp.Body)
		return returnMap, hash, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.New("cant read the body of the response: " + err.Error())
		return returnMap, hash, err
	}
	if err := json.Unmarshal([]byte(body), &arrSoftwareMaps); err != nil {
		err = errors.New("cant unmarshal the body of the response: " + err.Error())
		return returnMap, hash, err
	}
	recordsHash := Hash(arrSoftwareMaps)
	hash = fmt.Sprintf("%v", recordsHash)
	for _, v := range arrSoftwareMaps {
		assetIdentifier := iToS(v["package/publisher"]) + iToS(v["package/name"]) + iToS(v["package/version"])
		returnMap[assetIdentifier] = make(map[string]interface{})
		for field, value := range v {
			fieldId := strings.Replace(field, "package/", "", 1)
			if fieldId == "first_installation" {
				t, _ := time.Parse("2006-01-02T15:04:05", iToS(value))
				value = t.Format("2006-01-02 15:04:05")
			}
			returnMap[assetIdentifier][fieldId] = value
		}
	}
	return returnMap, hash, err
}
