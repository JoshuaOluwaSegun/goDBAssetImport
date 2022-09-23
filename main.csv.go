package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

type customReader struct{ r io.Reader }

func (r *customReader) Read(b []byte) (n int, err error) {
	x := make([]byte, len(b))
	if n, err = r.r.Read(x); err != nil {
		return n, err
	}
	copy(b, bytes.Replace(x, []byte("\r"), []byte("\n"), -1))
	return n, nil
}

// func getAssetsFromCSV(csvFile string, assetType assetTypesStruct) (bool, []map[string]string) {
func getAssetsFromCSV(assetType assetTypesStruct) (bool, map[string]map[string]interface{}) {
	//Initialise Asset Map
	arrAssetMaps := make(map[string]map[string]interface{})
	logger(3, " ", false, false)
	logger(3, "Running CSV query for "+assetType.AssetType+" assets. Please wait...", true, true)

	file, err := os.Open(assetType.CSVFile)
	if err != nil {
		// elements passed are separated by space automatically
		logger(4, "Error opening CSV file: "+err.Error()+" for "+assetType.AssetType+" assets.", true, true)
		return false, arrAssetMaps
	}
	defer file.Close()

	bom := make([]byte, 3)
	file.Read(bom)
	if bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		// BOM Detected, continue with feeding the file
	} else {
		// No BOM Detected, reset the file feed
		file.Seek(0, 0)
	}

	var r *csv.Reader
	if importConf.SourceConfig.CSV.CarriageReturnRemoval {
		custom := &customReader{file}
		r = csv.NewReader(custom)
	} else {
		r = csv.NewReader(file)
	}
	//because the json configuration loader cannot handle runes, code here to convert string to rune-array and getting first item
	if importConf.SourceConfig.CSV.CommaCharacter != "" {
		CSVCommaRunes := []rune(importConf.SourceConfig.CSV.CommaCharacter)
		r.Comma = CSVCommaRunes[0]
	}

	if importConf.SourceConfig.CSV.LazyQuotes {
		r.LazyQuotes = true
	}
	if importConf.SourceConfig.CSV.FieldsPerRecord > 0 {
		r.FieldsPerRecord = importConf.SourceConfig.CSV.FieldsPerRecord
	}
	var header []string

	intAssetCount := 0
	intAssetSuccess := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger(4, "Error reading CSV data: "+err.Error()+" for "+assetType.AssetType+" assets.", true, true)
			return false, arrAssetMaps
		}
		if header == nil {
			header = record
		} else {
			intAssetCount++
			var dict = make(map[string]interface{})
			for i := range header {
				dict[header[i]] = record[i]
			}
			intAssetSuccess++
			arrAssetMaps[fmt.Sprintf("%s", dict[assetType.AssetIdentifier.SourceColumn])] = dict
		}
	}
	logger(3, ""+strconv.Itoa(intAssetSuccess)+" of "+strconv.Itoa(intAssetCount)+" returned assets successfully retrieved ready for processing.", true, true)
	return true, arrAssetMaps
}
