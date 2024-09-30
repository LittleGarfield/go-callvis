package main

import (
	"encoding/json"
	"fmt"
	"go/token"
	"golang.org/x/tools/go/ssa"
	"os"
	"strconv"
	"strings"
)

type FuncDetail struct {
	FileID        string         `json:"fileID"`
	Range         []int          `json:"range"`
	RtnType       string         `json:"rtnType"`
	QualifiedName string         `json:"qualifiedName"`
	Name          string         `json:"name"`
	Parameters    []Parameter    `json:"parameters"`
	ThisClass     string         `json:"thisClass"`
	BaseClass     []string       `json:"baseClass"`
	Sig           string         `json:"sig"`
	Loc           string         `json:"loc"`
	Calls         []string       `json:"calls"`
	CallPositions []CallPosition `json:"callPositions"`
	CalledBy      []int          `json:"CalledBy"`
}
type Parameter struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
type CallPosition struct {
	Range  []int  `json:"range"`
	FuncID string `json:"funcID"`
}

var funcDetailMap = make(map[string]FuncDetail)

func getIdByKey(data map[string]int, key string, idPtr *int) int {
	if id, exists := data[key]; exists {
		return id
	}
	*idPtr++
	data[key] = *idPtr
	return data[key]
}
func getFuncRange(fn *ssa.Function, fs *token.FileSet) []int {
	var posStart, posEnd token.Position
	if fn.Synthetic == "" { // 源码定义函数, 非合成函数
		posStart = fs.Position(fn.Syntax().Pos())
		posEnd = fs.Position(fn.Syntax().End())
	} else {
		posStart = fs.Position(fn.Pos())
		posEnd = fs.Position(fn.Pos())
	}
	return []int{posStart.Line, posStart.Column, posEnd.Line, posEnd.Column}
}
func makeFuncDetailAndSave(fn *ssa.Function, fs *token.FileSet) {
	// 文件
	pathPrefix := *projDir
	if pathPrefix[len(pathPrefix)-1] != '/' { // 确保路径以 "/" 结尾
		pathPrefix += "/"
	}
	prefixLen := len(pathPrefix)
	file := fs.Position(fn.Pos()).Filename[prefixLen:] // 相对路径

	// 参数列表
	psStr := ""
	params := []Parameter{}
	for idx, p := range fn.Params {
		items := strings.Split(p.String(), " ")
		params = append(params, Parameter{
			Type: items[3],
			Name: items[1],
		})
		psStr += items[3]
		if idx < len(fn.Params)-1 {
			psStr += ","
		}
	}

	// 返回值
	retType := fn.Signature.Results().String()

	funcDetailMap[fn.String()] = FuncDetail{
		FileID:        file,
		Range:         getFuncRange(fn, fs),
		RtnType:       retType[1 : len(retType)-1],
		QualifiedName: fn.String(),
		Name:          fn.Name(),
		Parameters:    params,
		ThisClass:     "",
		BaseClass:     []string{},
		Sig:           fmt.Sprintf("%s(%s)", fn.String(), psStr),
		Loc:           "",
		Calls:         []string{},
		CallPositions: []CallPosition{},
	}
}
func makeFuncInfoAndSave(edges []*dotEdge) {

	// 更新 Calls callPositions 字段 TODO
	for _, edge := range edges {
		from := edge.From.ID
		to := edge.To.ID
		tps := strings.Split(edge.Attrs["tooltip"], ":")
		dtl := funcDetailMap[from]

		// Calls
		dtl.Calls = append(dtl.Calls, to)

		// CallPositions
		row := 0
		col := 0
		if val, err := strconv.Atoi(tps[1]); err == nil {
			row = val
		}
		if val, err := strconv.Atoi(tps[2]); err == nil {
			col = val
		}
		dtl.CallPositions = append(dtl.CallPositions, CallPosition{
			Range:  []int{row, col, row, col},
			FuncID: to,
		})
		funcDetailMap[from] = dtl
	}

	writeJson(funcDetailMap, fmt.Sprintf("%s.json", *outputFile))
}
func writeJson(data any, outFile string) bool {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return false
	}
	// 创建或打开文件
	file, err := os.Create(outFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return false
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error close file:", err)
		}
	}(file)
	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return false
	}
	fmt.Println("JSON data written to", outFile)
	return true
}
