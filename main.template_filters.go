package main

import (
	"fmt"
	"strings"
	"text/template"
)

var (
	TemplateFilters template.FuncMap
)


func checkTemplate() bool {
	blnFoundError := false
	for k, v := range SQLImportConf.AssetGenericFieldMapping {
		str := fmt.Sprintf("%v", v)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [" + k + "]")
			blnFoundError = true
		}
	}
	for k, v := range SQLImportConf.AssetTypeFieldMapping {
		str := fmt.Sprintf("%v", v)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [" + k + "]")
			blnFoundError = true
		}
	}
	for _, assetType := range SQLImportConf.AssetTypes {
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
	}
}
