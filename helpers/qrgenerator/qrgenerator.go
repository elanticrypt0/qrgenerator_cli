package qrgenerator

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// OutputFormat define el tipo de formato de salida
type OutputFormat string

// Formatos soportados
const (
	FormatPNG  OutputFormat = "png"
	FormatJPEG OutputFormat = "jpeg"
	FormatSVG  OutputFormat = "svg"
	FormatCSS  OutputFormat = "css"
)

// QRConfig contiene la configuración para generar el código QR
type QRConfig struct {
	URL         string
	LogoPath    string            // Ruta al archivo de logo (opcional)
	Size        int               // Tamaño del QR en píxeles
	OutputPath  string            // Ruta de salida
	Format      OutputFormat      // Formato de salida
	ExtraParams map[string]string // Parámetros adicionales para formatos especiales
}

// QRGenerator interface define los métodos que debe implementar cada generador de formato
type QRGenerator interface {
	Generate(qrImage image.Image, config QRConfig) error
}

// Implementaciones específicas para cada formato
type pngGenerator struct{}
type jpegGenerator struct{}
type svgGenerator struct{}

// generateQRImage genera la imagen base del QR con o sin logo
func generateQRImage(config QRConfig) (image.Image, error) {
	if config.Size == 0 {
		config.Size = 256 // Tamaño por defecto
	}

	// Generar el código QR
	qr, err := qrcode.New(config.URL, qrcode.Highest)
	if err != nil {
		return nil, fmt.Errorf("error generando QR: %w", err)
	}

	// Generar la imagen del QR
	qrImage := qr.Image(config.Size)

	// // Si hay un logo, procesarlo y superponerlo
	// TODO
	// if config.LogoPath != "" {
	// 	err = overlayLogo(qrImage, config.LogoPath, config.Size)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("error superponiendo logo: %w", err)
	// 	}
	// }

	return qrImage, nil
}

// overlayLogo superpone un logo en el centro del QR
func overlayLogo(qrImage *image.RGBA, logoPath string, size int) error {
	var logoImg image.Image
	ext := filepath.Ext(logoPath)

	switch strings.ToLower(ext) {
	case ".svg":
		icon, err := oksvg.ReadIcon(logoPath, oksvg.StrictErrorMode)
		if err != nil {
			return fmt.Errorf("error leyendo SVG: %w", err)
		}

		logoSize := int(float64(size) * 0.3)
		icon.SetTarget(0, 0, float64(logoSize), float64(logoSize))

		rgba := image.NewRGBA(image.Rect(0, 0, logoSize, logoSize))
		scanner := rasterx.NewScannerGV(logoSize, logoSize, rgba, rgba.Bounds())
		raster := rasterx.NewDasher(logoSize, logoSize, scanner)
		icon.Draw(raster, 1.0)
		logoImg = rgba

	case ".png", ".jpg", ".jpeg":
		f, err := os.Open(logoPath)
		if err != nil {
			return fmt.Errorf("error abriendo imagen: %w", err)
		}
		defer f.Close()

		logoImg, _, err = image.Decode(f)
		if err != nil {
			return fmt.Errorf("error decodificando imagen: %w", err)
		}

	default:
		return fmt.Errorf("formato de logo no soportado: %s", ext)
	}

	// Calcular posición central
	logoSize := int(float64(size) * 0.3)
	offset := (size - logoSize) / 2
	logoRect := image.Rect(offset, offset, offset+logoSize, offset+logoSize)

	draw.Draw(qrImage, logoRect, logoImg, image.Point{}, draw.Over)
	return nil
}

// Implementación para PNG
func (g *pngGenerator) Generate(qrImage image.Image, config QRConfig) error {
	f, err := os.Create(config.OutputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo PNG: %w", err)
	}
	defer f.Close()

	quality := 100
	if qualityStr, ok := config.ExtraParams["quality"]; ok {
		// Parsear calidad si está especificada
		fmt.Sscanf(qualityStr, "%d", &quality)
	}

	enc := &png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	return enc.Encode(f, qrImage)
}

// Implementación para JPEG
func (g *jpegGenerator) Generate(qrImage image.Image, config QRConfig) error {
	f, err := os.Create(config.OutputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo JPEG: %w", err)
	}
	defer f.Close()

	quality := 90
	if qualityStr, ok := config.ExtraParams["quality"]; ok {
		fmt.Sscanf(qualityStr, "%d", &quality)
	}

	return jpeg.Encode(f, qrImage, &jpeg.Options{Quality: quality})
}

// Implementación para SVG
func (g *svgGenerator) Generate(qrImage image.Image, config QRConfig) error {
	f, err := os.Create(config.OutputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo SVG: %w", err)
	}
	defer f.Close()

	// Convertir la imagen a una representación SVG
	bounds := qrImage.Bounds()
	svgContent := bytes.Buffer{}

	svgContent.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
		<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">
		<rect width="100%%" height="100%%" fill="white"/>`,
		bounds.Dx(), bounds.Dy(), bounds.Dx(), bounds.Dy()))

	// Convertir píxeles a rectángulos SVG
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			color := qrImage.At(x, y)
			r, g, b, a := color.RGBA()
			if a > 0 && r == 0 && g == 0 && b == 0 { // Solo dibujar píxeles negros
				svgContent.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="black"/>`, x, y))
			}
		}
	}

	svgContent.WriteString("</svg>")
	_, err = f.Write(svgContent.Bytes())
	return err
}

// Implementación del generador CSS
type cssGenerator struct{}

func (g *cssGenerator) Generate(qrImage image.Image, config QRConfig) error {
	f, err := os.Create(config.OutputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo CSS: %w", err)
	}
	defer f.Close()

	bounds := qrImage.Bounds()
	var cssContent bytes.Buffer

	// Escribir el CSS base
	cssContent.WriteString(`
.qr-code {
    width: 1px;
    height: 1px;
    position: relative;
    background: white;
    box-shadow: `)

	// Variables para tracking
	var shadows []string
	pixelSize := 1
	if size, ok := config.ExtraParams["pixel-size"]; ok {
		fmt.Sscanf(size, "%d", &pixelSize)
	}

	// Generar box-shadows para cada pixel negro
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			color := qrImage.At(x, y)
			r, g, b, a := color.RGBA()
			if a > 0 && r == 0 && g == 0 && b == 0 { // Solo pixeles negros
				shadow := fmt.Sprintf("%dpx %dpx 0 %dpx black",
					x*pixelSize,
					y*pixelSize,
					pixelSize/2)
				shadows = append(shadows, shadow)
			}
		}
	}

	// Unir todos los box-shadows
	cssContent.WriteString(strings.Join(shadows, ",\n    "))
	cssContent.WriteString(";\n}\n\n")

	// Agregar reglas de tamaño y centrado
	cssContent.WriteString(fmt.Sprintf(`
.qr-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    background: white;
    padding: 20px;
}

.qr-code {
    transform: scale(%d);
    margin: %dpx;
}`,
		pixelSize,
		bounds.Dx()*pixelSize/2))

	// Agregar HTML de ejemplo si está configurado
	if includeHTML, ok := config.ExtraParams["include-html"]; ok && includeHTML == "true" {
		cssContent.WriteString(`

<!-- Ejemplo de uso -->
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>CSS QR Code</title>
    <style>
        /* Pegar el CSS anterior aquí */
    </style>
</head>
<body>
    <div class="qr-container">
        <div class="qr-code"></div>
    </div>
</body>
</html>
`)
	}

	_, err = f.Write(cssContent.Bytes())
	return err
}

// GenerateQR es la función principal que genera el código QR en el formato especificado
func GenerateQR(config QRConfig) error {
	// Validar configuración
	if config.URL == "" {
		return fmt.Errorf("URL es requerida")
	}

	if config.ExtraParams == nil {
		config.ExtraParams = make(map[string]string)
	}

	// Generar la imagen base del QR
	qrImage, err := generateQRImage(config)
	if err != nil {
		return err
	}

	// Seleccionar el generador según el formato
	var generator QRGenerator
	switch config.Format {
	case FormatPNG:
		generator = &pngGenerator{}
	case FormatJPEG:
		generator = &jpegGenerator{}
	case FormatSVG:
		generator = &svgGenerator{}
	case FormatCSS:
		generator = &cssGenerator{}
	default:
		return fmt.Errorf("formato no soportado: %s", config.Format)
	}

	// Generar el archivo de salida
	return generator.Generate(qrImage, config)
}
