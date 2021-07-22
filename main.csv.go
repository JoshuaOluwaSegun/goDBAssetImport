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

//func getAssetsFromCSV(csvFile string, assetType assetTypesStruct) (bool, []map[string]string) {
func getAssetsFromCSV(csvFile string, assetType assetTypesStruct) (bool, map[string]map[string]interface{}) {
	//Initialise Asset Map
	arrAssetMaps := make(map[string]map[string]interface{})
	logger(1, " ", false, false)
	logger(3, "[CSV] Running CSV query for "+assetType.AssetType+" assets. Please wait...", true, true)

	//rows := []map[string]string{}
	file, err := os.Open(csvFile)
	if err != nil {
		// err is printable
		// elements passed are separated by space automatically
		logger(4, "[CSV] Error opening CSV file: "+err.Error()+" for "+assetType.AssetType+" assets.", true, true)
		return false, arrAssetMaps
	}
	// automatically call Close() at the end of current method
	defer file.Close()

	bom := make([]byte, 3)
	file.Read(bom)
	if bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		// BOM Detected, continue with feeding the file fmt.Println("BOM")
	} else {
		// No BOM Detected, reset the file feed
		file.Seek(0, 0)
	}

	var r *csv.Reader
	if SQLImportConf.CSVConf.CarriageReturnRemoval {
		custom := &customReader{file}
		r = csv.NewReader(custom)
	} else {
		r = csv.NewReader(file)
	}
	//because the json configuration loader cannot handle runes, code here to convert string to rune-array and getting first item
	if SQLImportConf.CSVConf.CommaCharacter != "" {
		CSVCommaRunes := []rune(SQLImportConf.CSVConf.CommaCharacter)
		r.Comma = CSVCommaRunes[0]
		//r.Comma = ';'
	}

	if SQLImportConf.CSVConf.LazyQuotes {
		r.LazyQuotes = true
	}
	if SQLImportConf.CSVConf.FieldsPerRecord > 0 {
		r.FieldsPerRecord = SQLImportConf.CSVConf.FieldsPerRecord
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
			logger(4, "[CSV] Error reading CSV data: "+err.Error()+" for "+assetType.AssetType+" assets.", true, true)
			return false, arrAssetMaps
		}
		if header == nil {
			header = record
		} else {
			intAssetCount++
			//dict := map[string]string{}
			var dict = make(map[string]interface{})
			for i := range header {
				dict[header[i]] = record[i]
			}
			intAssetSuccess++
			arrAssetMaps[fmt.Sprintf("%s", dict[assetType.AssetIdentifier.DBColumn])] = dict
			//rows = append(rows, dict)
		}
	}
	logger(3, "[CSV] "+strconv.Itoa(intAssetSuccess)+" of "+strconv.Itoa(intAssetCount)+" returned assets successfully retrieved ready for processing.", true, true)

	return true, arrAssetMaps

}
