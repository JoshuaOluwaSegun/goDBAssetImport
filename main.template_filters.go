package main

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	TemplateFilters template.FuncMap
)

func checkTemplate() bool {
	blnFoundError := false
	for k, v := range importConf.AssetGenericFieldMapping {
		str := fmt.Sprintf("%v", v)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [" + k + "]")
			blnFoundError = true
		}
	}
	for k, v := range importConf.AssetTypeFieldMapping {
		str := fmt.Sprintf("%v", v)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [" + k + "]")
			blnFoundError = true
		}
	}
	for _, assetType := range importConf.AssetTypes {
		for k, v := range assetType.SoftwareInventory.Mapping {
			str := fmt.Sprintf("%v", v)
			t := template.New(str).Funcs(TemplateFilters)
			_, err := t.Parse(str)
			if err != nil {
				fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [" + assetType.AssetType + "." + k + "]")
				blnFoundError = true
			}
		}
	}
	return blnFoundError
}

func setTemplateFilters() {
	TemplateFilters = template.FuncMap{
		"Upper": func(feature string) string {
			return strings.ToUpper(feature)
		},
		"Lower": func(feature string) string {
			return strings.ToLower(feature)
		},
		"epoch": func(feature string) string {
			result := ""
			if feature == "" || feature == "0" {
			} else {
				t, err := strconv.ParseInt(feature, 10, 0)
				if err == nil {
					md := time.Unix(t, 0)
					result = md.Format("2006-01-02 15:04:05")
				}
			}
			return result
		},
		"epoch_clear": func(feature string) string {
			result := "__clear__"
			if feature == "" || feature == "0" {
			} else {
				t, err := strconv.ParseInt(feature, 10, 0)
				if err == nil {
					md := time.Unix(t, 0)
					result = md.Format("2006-01-02 15:04:05")
				}
			}
			return result
		},
		// {{ .Name | date_conversion \"02/01/2006 15:04\" }}
		"date_conversion": func(layoutFormat string, dateString string) string {
			result := ""
			dt, err := time.Parse(layoutFormat, dateString)
			if err == nil {
				result = dt.Format("2006-01-02 15:04:05")
			}
			return result
		},
		"date_conversion_clear": func(layoutFormat string, dateString string) string {
			result := "__clear__"
			dt, err := time.Parse(layoutFormat, dateString)
			if err == nil {
				result = dt.Format("2006-01-02 15:04:05")
			}
			return result
		},
	}
}
