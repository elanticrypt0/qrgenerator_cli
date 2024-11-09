package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"qrgenerator_cli/helpers/qrgenerator"
)

func main() {

	qr_url := flag.String("url", "https://tryhackme.com", "Url to go with QR")
	qr_size := flag.Int("size", 256, "QR size")
	qr_output := flag.String("o", "new_qr.jpg", "Output path and file with extension. Formats: jpg, png, svg, css")

	flag.Parse()

	qr_type := filepath.Ext(*qr_output)

	var qr_format_type qrgenerator.OutputFormat

	switch qr_type[1:] {
	case "jpg":
		qr_format_type = qrgenerator.FormatJPEG
	case "png":
		qr_format_type = qrgenerator.FormatPNG
	case "svg":
		qr_format_type = qrgenerator.FormatSVG
	case "css":
		qr_format_type = qrgenerator.FormatCSS
	default:
		qr_format_type = qrgenerator.FormatJPEG
	}

	config := qrgenerator.QRConfig{
		URL:        *qr_url,
		Size:       *qr_size,
		OutputPath: *qr_output,
		Format:     qr_format_type,
	}
	err := qrgenerator.GenerateQR(config)
	if err != nil {
		log.Printf("%q", err)
	}

	fmt.Println("QR Generator")
	fmt.Println("> Configuracion")
	fmt.Printf("%v", config)
}
