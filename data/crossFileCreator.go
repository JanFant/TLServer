package data

import (
	"../logger"
	u "../utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

//SelectedData
type SelectedData struct {
	SelectedData map[string]map[string][]CheckData `json:"selected"`
	PngSettings  PngSettings                       `json:"pngSettings"`
}

//CheckData структура проверки для перекрестков
type CheckData struct {
	ID        string `json:"ID"`
	PngStatus bool   `json:"pngStatus"`
	SvgStatus bool   `json:"svgStatus"`
}

//PngSettings настройки размеров создаваемой map.png
type PngSettings struct {
	SizeX string `json:"sizeX"`
	SizeY string `json:"sizeY"`
	Z     string `json:"z"`
}

func (checkData *CheckData) setStatusTrue() {
	checkData.SvgStatus = true
	checkData.PngStatus = true
}

func (checkData *CheckData) setStatusFalse() {
	checkData.SvgStatus = false
	checkData.PngStatus = false
}

//stockData заполняет поля из env файла
func (set *PngSettings) stockData() {
	set.SizeX = os.Getenv("png_sizeX")
	set.SizeY = os.Getenv("png_sizeY")
	set.Z = os.Getenv("png_Z")
}

//MainCrossCreator формираю необходимые данные для начальной странички с деревом
func MainCrossCreator() map[string]interface{} {
	//CacheInfoDataUpdate()
	tfData := GetAllTrafficLights()
	mapRegAreaCross := make(map[string]map[string][]TrafficLights)
	for numReg, nameReg := range CacheInfo.mapRegion {
		if strings.Contains(numReg, "*") {
			continue
		}
		for numArea, nameArea := range CacheInfo.mapArea[nameReg] {
			if strings.Contains(numArea, "Все регионы") {
				continue
			}
			if _, ok := mapRegAreaCross[nameReg]; !ok {
				mapRegAreaCross[nameReg] = make(map[string][]TrafficLights)
			}
			for _, tempTF := range tfData {
				if (tempTF.Region.Num == numReg) && (tempTF.Area.Num == numArea) {
					mapRegAreaCross[nameReg][nameArea] = append(mapRegAreaCross[nameReg][nameArea], tempTF)
				}
			}

		}
	}
	var settings PngSettings
	settings.stockData()

	resp := u.Message(true, "Cross creator main page, formed a crosses map")
	resp["crossMap"] = mapRegAreaCross
	resp["pngSettings"] = settings
	resp["region"] = CacheInfo.mapRegion
	resp["area"] = CacheInfo.mapArea
	return resp
}

//CheckCrossDirFromBD проверяет ВСЕ перекрестки из БД на наличие каталого для них и заполнения
func CheckCrossDirFromBD() map[string]interface{} {
	//CacheInfoDataUpdate()
	tfData := GetAllTrafficLights()
	path := os.Getenv("views_path") + "//cross"
	var tempTF []TrafficLights
	for _, tfLight := range tfData {
		_, err1 := os.Stat(path + fmt.Sprintf("//%v//%v//%v//map.png", tfLight.Region.Num, tfLight.Area.Num, tfLight.ID))
		_, err2 := os.Stat(path + fmt.Sprintf("//%v//%v//%v//cross.svg", tfLight.Region.Num, tfLight.Area.Num, tfLight.ID))
		if os.IsNotExist(err1) || os.IsNotExist(err2) {
			tempTF = append(tempTF, tfLight)
		}
	}
	resp := make(map[string]interface{})
	resp["tf"] = tempTF
	return resp
}

//CheckCrossDirSelected проверяет региона/районы/перекрестки которые запросил пользователь
func CheckCrossFileSelected(selectedData map[string]map[string][]CheckData) map[string]interface{} {
	path := os.Getenv("views_path") + "//cross"
	for numFirst, firstMap := range selectedData {
		for numSecond, secondMap := range firstMap {
			for numCheck, check := range secondMap {
				_, err1 := os.Stat(path + fmt.Sprintf("//%v//%v//%v//map.png", numFirst, numSecond, check.ID))
				if !os.IsNotExist(err1) {
					selectedData[numFirst][numSecond][numCheck].PngStatus = true
				}
				_, err2 := os.Stat(path + fmt.Sprintf("//%v//%v//%v//cross.svg", numFirst, numSecond, check.ID))
				if !os.IsNotExist(err2) {
					selectedData[numFirst][numSecond][numCheck].SvgStatus = true
				}
			}
		}
	}
	resp := make(map[string]interface{})
	resp["verifiedData"] = selectedData
	return resp
}

//MakeSelectedDir создание каталогов и файлов png + svg у выбранных
func MakeSelectedDir(selData SelectedData) map[string]interface{} {
	var (
		message []string
		count   = 0
	)
	sizeX, _ := strconv.Atoi(selData.PngSettings.SizeX)
	sizeY, _ := strconv.Atoi(selData.PngSettings.SizeY)
	if selData.PngSettings.SizeX == "" || selData.PngSettings.SizeY == "" || selData.PngSettings.Z == "" || sizeX > 450 || sizeX < 0 || sizeY < 0 || sizeY > 450 {
		selData.PngSettings.stockData()
	}
	path := os.Getenv("views_path") + "//cross"
	for numFirst, firstMap := range selData.SelectedData {
		for numSecond, secondMap := range firstMap {
			for numCheck, check := range secondMap {
				err := os.MkdirAll(path+fmt.Sprintf("//%v//%v//%v", numFirst, numSecond, check.ID), os.ModePerm)
				if err != nil {
					selData.SelectedData[numFirst][numSecond][numCheck].setStatusFalse()
					continue
				}
				if !selData.SelectedData[numFirst][numSecond][numCheck].PngStatus {
					point, err := TakePointFromBD(numFirst, numSecond, check.ID)
					if err != nil {
						logger.Error.Println(fmt.Sprintf("|Message: No result at point: (%v//%v//%v)", numFirst, numSecond, check.ID))
						if count == 0 {
							message = append(message, "Не созданны")
						}
						count++
						message = append(message, fmt.Sprintf("%v: (%v//%v//%v)", count, numFirst, numSecond, check.ID))
						continue
					}
					err = createPng(numFirst, numSecond, check.ID, selData.PngSettings, point)
					if err != nil {
						logger.Error.Println(fmt.Sprintf("|Message: Can't create map.png path = %v//%v//%v", numFirst, numSecond, check.ID))
						continue
					}
					selData.SelectedData[numFirst][numSecond][numCheck].PngStatus = true
				}
			}
		}
	}
	resp := make(map[string]interface{})
	if len(message) == 0 {
		resp = u.Message(true, "Everything was created")
	} else {
		resp = u.Message(false, "There are not created")
		resp["notCreated"] = message
	}
	resp["makeData"] = selData
	return resp
}

func ShortCreateDirPng(region, area, id int, pointStr string) bool {
	var (
		pngSettings PngSettings
		point       Point
	)
	pngSettings.stockData()
	point.StrToFloat(pointStr)
	path := os.Getenv("views_path") + "//cross"
	_ = os.MkdirAll(path+fmt.Sprintf("//%v//%v//%v", region, area, id), os.ModePerm)
	err := createPng(strconv.Itoa(region), strconv.Itoa(area), strconv.Itoa(id), pngSettings, point)
	if err != nil {
		logger.Error.Println(fmt.Sprintf("|Message: Can't create map.png path = %v//%v//%v", region, area, id))
		return false
	}
	_, err = os.Stat(path + fmt.Sprintf("//%v//%v//%v//cross.svg", region, area, id))
	if os.IsNotExist(err) {
		file1, err := os.Create(path + fmt.Sprintf("//%v//%v//%v//cross.svg", region, area, id))
		if err != nil {
			return false
		}
		defer file1.Close()
		_, err = fmt.Fprintln(file1, str1, str2)
		if err != nil {
			return false
		}
	}
	return true
}

var (
	str1 = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg
			xmlns:svg="http://www.w3.org/2000/svg"
			xmlns="http://www.w3.org/2000/svg"
			width="450mm"
			height="450mm"
			viewBox="0 0 450 450">
			<foreignObject x="5" y="5" width="100" height="450">
			<div xmlns="http://www.w3.org/1999/xhtml"
		style="font-size:8px;font-family:sans-serif">`
	str2 = `</div>
			</foreignObject>
 			</svg>`
)

//createPng создание png файла
func createPng(numReg, numArea, id string, settings PngSettings, point Point) (err error) {
	url := fmt.Sprintf("https://static-maps.yandex.ru/1.x/?ll=%3.15f,%3.15f&z=%v&l=map&size=%v,%v", point.X, point.Y, settings.Z, settings.SizeX, settings.SizeY)
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	filePath := os.Getenv("views_path") + "//cross" + "//" + numReg + "//" + numArea + "//" + id + "//"
	//open a file for writing
	file, err := os.Create(filePath + "map.png")
	if err != nil {
		return err
	}
	defer file.Close()
	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func createSvg(path string) (err error) {

	//file1, err := os.Create(filepath + "cross.svg")
	//if err != nil {
	//	return err
	//}
	//defer file1.Close()
	//str3 := fmt.Sprintf("%s", TL.Description)
	//fmt.Fprintln(file1, str1, str3, str2)

	return err
}
