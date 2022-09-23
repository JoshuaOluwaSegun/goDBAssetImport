package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type workspaceOneTokenStruct struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type workspaceOneResponseStruct struct {
	Applications []map[string]interface{} `json:"app_items"`
	Devices      []map[string]interface{} `json:"Devices"`
	Page         int64                    `json:"Page"`
	PageSize     int64                    `json:"PageSize"`
	Total        int64                    `json:"Total"`
}

func getAssetsFromWorkspaceOne(assetType assetTypesStruct) (map[string]map[string]interface{}, error) {
	//Initialise Asset Map
	returnMap := make(map[string]map[string]interface{})
	logger(3, " ", false, false)
	logger(3, "[WORKSPACEONE] Running VMWare Workspace One UEM query for "+assetType.AssetType+" assets. Please wait...", true, true)

	pageURL := key.Domain + "/API/mdm/devices/search"
	filterAdded := false
	v := reflect.ValueOf(assetType.Filters)
	typeOfS := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).String() != "" {
			if !filterAdded {
				pageURL += "?"
			} else {
				pageURL += "&"
			}
			filterAdded = true
			pageURL += strings.ToLower(typeOfS.Field(i).Name) + "=" + url.PathEscape(v.Field(i).String())
		}
	}
	pageNum := 0
	for {
		assetsList, err := getDevicesPageWorkspaceOne(assetType, pageURL, pageNum, filterAdded)
		if err != nil {
			return returnMap, err
		}
		for _, v := range assetsList.Devices {
			//Get the asset ID for the current record
			assetIDIdent := fmt.Sprintf("%v", assetType.AssetIdentifier.SourceColumn)
			t := template.New(assetIDIdent).Funcs(TemplateFilters)
			tmpl, _ := t.Parse(assetIDIdent)
			buf := bytes.NewBufferString("")
			tmpl.Execute(buf, v)
			assetIdentifier := ""
			if buf != nil {
				assetIdentifier = buf.String()
			}
			//Get installed software

			v["InstalledSoftware"], err = getInstalledAppsWorkspaceOne(v["Uuid"].(string), assetType)
			if err != nil {
				return returnMap, errors.New("error when retrieving apps list from Workspace One UEM: " + err.Error())
			}
			returnMap[assetIdentifier] = v
		}
		pageNum++
		if len(assetsList.Devices) == 0 {
			break
		}
	}
	return returnMap, nil
}

func getInstalledAppsWorkspaceOne(deviceUUID string, assetType assetTypesStruct) ([]map[string]interface{}, error) {
	var (
		installedApps []map[string]interface{}
		err           error
		pageNum       = 0
		pageURL       = key.Domain + "/API/mdm/devices/" + deviceUUID + "/apps/search"
	)
	logger(3, "[WORKSPACEONE] Running VMWare Workspace One UEM query for apps installed on "+deviceUUID+". Please wait...", false, true)

	for {
		appsList, err := getAppsPageWorkspaceOne(assetType, pageURL, pageNum)
		if err != nil {
			return installedApps, err
		}
		if len(appsList.Applications) == 0 {
			break
		}
		installedApps = append(installedApps, appsList.Applications...)
		pageNum++
	}
	return installedApps, err
}

func getAppsPageWorkspaceOne(assetType assetTypesStruct, pageURL string, pageNum int) (appsResponse workspaceOneResponseStruct, err error) {
	currPageURL := pageURL + "?page=" + strconv.Itoa(pageNum)
	logger(2, "Getting page of apps on from VMWare Workspace One UEM, URL: "+currPageURL, false, true)
	req, err := http.NewRequest("GET", currPageURL, nil)

	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+key.AccessToken)
	req.Header.Set("User-Agent", appName+"/"+version)
	req.Header.Set("Accept", "application/json;version=3")

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}, Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	//-- Check for HTTP Response
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(io.Discard, resp.Body)
		return
	}
	if resp.StatusCode == 204 {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return appsResponse, errors.New("Failed to read the body of the response: " + err.Error())
	}
	if err := json.Unmarshal([]byte(body), &appsResponse); err != nil {
		return appsResponse, errors.New("Failed to unmarshal JSON response from Workforce One UEM: " + err.Error())
	}
	return
}

func getDevicesPageWorkspaceOne(assetType assetTypesStruct, pageURL string, pageNum int, filtered bool) (assetsResponse workspaceOneResponseStruct, err error) {
	currPageURL := pageURL
	if filtered {
		currPageURL += "&"
	} else {
		currPageURL += "?"
	}
	currPageURL += "page=" + strconv.Itoa(pageNum)
	logger(2, "Getting page of assets from VMWare Workspace One UEM, URL: "+currPageURL, false, true)
	req, err := http.NewRequest("GET", currPageURL, nil)

	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+key.AccessToken)
	req.Header.Set("User-Agent", appName+"/"+version)
	req.Header.Set("Accept", "application/json;version=3")

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}, Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	//-- Check for HTTP Response
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(io.Discard, resp.Body)
		return
	}
	if resp.StatusCode == 204 {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return assetsResponse, errors.New("Failed to read the body of the response: " + err.Error())
	}
	if err := json.Unmarshal([]byte(body), &assetsResponse); err != nil {
		return assetsResponse, errors.New("Failed to unmarshal JSON response from Workforce One UEM: " + err.Error())
	}
	return
}

func generateWorkspaceOneAccessToken() (tokenResponse workspaceOneTokenStruct, err error) {
	formURL := "https://" + key.Region + ".uemauth.vmwservices.com/connect/token"
	formPayload := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {key.ClientID},
		"client_secret": {key.ClientSecret},
	}
	resp, err := http.PostForm(formURL, formPayload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(body), &tokenResponse)
	return
}
