package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/manifoldco/promptui"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

var (
	sourceDir      string
	destinationDir string
	data           map[string]string
)

func readJson(fp string) (map[string]string, error) {
	jsonFile, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)

	// Unmarshal the JSON to the 'data' variable
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func RenderTemplates(srcDir string, destDir string, data map[string]string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() || path == filepath.Join(srcDir, "config.json") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse the file content as a template
		tmpl, err := template.New(path).Parse(string(content))
		if err != nil {
			return err
		}

		// Create the output file, preserving the directory structure
		relPath, _ := filepath.Rel(srcDir, path)

		// Parse the relative path as a template
		pathTmpl, err := template.New("path").Parse(relPath)
		if err != nil {
			return err
		}

		// Execute the path template with the provided data
		var newPath bytes.Buffer
		if err := pathTmpl.Execute(&newPath, data); err != nil {
			return err
		}

		outPath := filepath.Join(destDir, newPath.String())
		outDir := filepath.Dir(outPath)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		// Execute the template with the provided data and write the result to the output file
		return tmpl.Execute(outFile, data)
	})
}

func getConfig(fp string) (map[string]string, error) {
	defaults, err := readJson(fp)
	if err != nil {
		fmt.Println("unable to read config.json from template directory")
		return nil, err
	}

	// Define a map to hold the actual values
	values := make(map[string]string)

	// Prompt the user for each input
	for key, defaultValue := range defaults {
		prompt := promptui.Prompt{
			Label:     key,
			Default:   defaultValue,
			AllowEdit: true,
		}

		result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return nil, err
		}
		// Store the value
		values[key] = result
	}

	return values, nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		os.Exit(1)
	}

	flag.StringVar(&sourceDir, "template", "", "The source directory containing the template files")
	flag.StringVar(&destinationDir, "destination", cwd, "The destination directory to write the rendered files")
	flag.Parse()

	if sourceDir == "" || destinationDir == "" {
		fmt.Println("Both source and destination directories are required")
		os.Exit(1)
	}

	config, err := getConfig(sourceDir + "/config.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = RenderTemplates(sourceDir, destinationDir, config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
