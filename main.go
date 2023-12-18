package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func fileIsExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func CmdToPdf(dir string, entry os.DirEntry, outDir string) {
	if entry.IsDir() {
		return
	}
	file, err := os.ReadFile(dir + "\\" + entry.Name())
	if err != nil {
		return
	}
	ext := filepath.Ext(entry.Name())
	saveFile, err := ioutil.TempFile("", entry.Name()+ext)
	if err != nil {
		log.Println("文件类型获取失败")
		return
	}

	defer os.Remove(saveFile.Name())
	defer saveFile.Close()
	reader := bytes.NewReader(file)
	_, err = io.Copy(saveFile, reader)
	if err != nil {
		log.Println(err)
		log.Println("アップロードファイルの書き込みに失敗しました。")
		return
	}

	word := new(Word)
	log.Println("input file: " + saveFile.Name())
	log.Println("output dir: " + outDir)
	//PDF変換
	outFilePath, err := word.Export(saveFile.Name(), outDir)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("output file: " + outFilePath)
}

func export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("POST Only"))
		return
	}

	file, fileInfo, err := r.FormFile("file")

	if err != nil {
		log.Println("ファイルアップロードを確認できませんでした。")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer file.Close()

	ext := filepath.Ext(fileInfo.Filename)
	saveFile, err := ioutil.TempFile("", "w2p*"+ext)
	if err != nil {
		log.Println("サーバ側でファイル確保できませんでした。")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer os.Remove(saveFile.Name())
	defer saveFile.Close()

	_, err = io.Copy(saveFile, file)
	if err != nil {
		log.Println(err)
		log.Println("アップロードファイルの書き込みに失敗しました。")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	outDir, _ := ioutil.TempDir("", "w2p")
	if err != nil {
		log.Println(err)
		log.Println("一時ディレクトリの作成に失敗しました。")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(outDir)

	word := new(Word)
	log.Println("input file: " + saveFile.Name())
	log.Println("output dir: " + outDir)

	//PDF変換
	outFilePath, err := word.Export(saveFile.Name(), outDir)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("output file: " + outFilePath)

	outFile, err := ioutil.ReadFile(outFilePath)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(outFile)))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(outFilePath)+`"`)
	w.Write(outFile)
}

func root(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">
    <html>
    <body>
    <form ENCTYPE="multipart/form-data" method="post" action="/upload">
    <input name="file" type="file"/>
    <input type="submit" value="upload"/>
    </form>
    </body>
    </html>
	`
	fmt.Fprintf(w, html)
}

func main() {

	var dir, fileName string
	flag.StringVar(&dir, "d", "", "路径")
	flag.StringVar(&fileName, "n", "", "文件名")
	flag.Parse()
	fmt.Println(dir, fileName)
	if dir != "" {
		readDir, err := os.ReadDir(dir)
		if err != nil {
			log.Println("os.ReadDir err " + err.Error())
			return
		}
		var outDir string
		for i := 0; i < 100; i++ {
			err := os.Mkdir(dir+"_out"+fmt.Sprint(i), os.ModeDir)
			if err != nil {
				continue
			} else {
				outDir = dir + "_out" + fmt.Sprint(i)
				break
			}
		}
		for _, file := range readDir {
			name := file.Name()
			if strings.HasSuffix(name, ".doc") || strings.HasSuffix(name, ".docx") {
				fmt.Println("to pdf", name)
				CmdToPdf(dir, file, outDir)
			}
		}
	}
	port := "8000"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	http.HandleFunc("/", root)
	http.HandleFunc("/upload", export)
	log.Println("Server is listening on port " + port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
