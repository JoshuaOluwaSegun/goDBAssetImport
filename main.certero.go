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
	"reflect"
	"strconv"
	"text/template"
	"time"
)

type certeroResponseStruct struct {
	Odata_context  string `json:"@odata.context"`
	Odata_nextLink string `json:"@odata.nextLink"`
	Assets         []struct {
		ClientProducts                int                      `json:"ClientProducts"`
		ClientType                    int                      `json:"ClientType"`
		ClientTypes                   int                      `json:"ClientTypes"`
		ComputerSystemConfigurationID int                      `json:"ComputerSystemConfigurationId"`
		ComputerSystemInventory       map[string]interface{}   `json:"ComputerSystemInventory"`
		ComputerSystemObjectID        int                      `json:"ComputerSystemObjectId"`
		ComputerSystemProcessorInfo   map[string]interface{}   `json:"ComputerSystemProcessorInfo"`
		Modules                       int                      `json:"Modules"`
		NodeObjectID                  int                      `json:"NodeObjectId"`
		OperatingSystem               int                      `json:"OperatingSystem"`
		WindowsSystemSoftwareProducts []map[string]interface{} `json:"WindowsSystemSoftwareProducts"`
	} `json:"value"`
}

func getAssetsFromCertero(assetType assetTypesStruct) (map[string]map[string]interface{}, error) {
	//Initialise Asset Map
	returnMap := make(map[string]map[string]interface{})
	logger(3, " ", false, false)
	logger(3, "[CERTERO] Running Certero query for "+assetType.AssetType+" assets. Please wait...", true, true)
	if importConf.SourceConfig.Certero.PageSize == 0 {
		importConf.SourceConfig.Certero.PageSize = 100
	}
	nextPageURL := key.Endpoint + "/?$top=" + strconv.Itoa(importConf.SourceConfig.Certero.PageSize) + "&$expand=" + importConf.SourceConfig.Certero.Expand
	if assetType.Query != "" {
		nextPageURL += "&$filter=" + url.PathEscape(assetType.Query)
	}
	for {
		assetsList, err := getDevicesPageCertero(assetType, nextPageURL)
		if err != nil {
			return returnMap, err
		}
		for _, v := range assetsList.Assets {
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

			assetRecord := make(map[string]interface{})
			rV := reflect.ValueOf(v)
			typeOfS := rV.Type()
			for i := 0; i < rV.NumField(); i++ {
				assetRecord[typeOfS.Field(i).Name] = rV.Field(i).Interface()
			}
			returnMap[assetIdentifier] = assetRecord
		}
		// Break the loop if no token is returned
		if assetsList.Odata_nextLink == "" {
			break
		}
		nextPageURL = assetsList.Odata_nextLink
	}
	if len(returnMap) == 0 {
		logger(3, "No "+assetType.AssetType+" asset records returned from Certero - check your configuration!", true, true)
	} else {
		logger(2, "Total "+assetType.AssetType+" asset records returned from Certero: "+strconv.Itoa(len(returnMap)), true, true)
	}
	return returnMap, nil
}

func getDevicesPageCertero(assetType assetTypesStruct, nextPageURL string) (assetsResponse certeroResponseStruct, err error) {
	logger(2, "Getting page of assets from Certero, URL: "+nextPageURL, false, true)
	req, err := http.NewRequest("GET", nextPageURL, nil)

	if err != nil {
		return
	}
	auth := base64.StdEncoding.EncodeToString([]byte(key.APIKeyName + ":" + key.APIKey))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", appName+"/"+version)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}, Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(io.Discard, resp.Body)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return assetsResponse, errors.New("Failed to read the body of the response: " + err.Error())
	}
	if err := json.Unmarshal([]byte(body), &assetsResponse); err != nil {
		log.Fatal(err)
	}
	return
}
