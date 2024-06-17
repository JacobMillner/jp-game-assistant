package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/kbinani/screenshot"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Set up the application with the custom font
	a := app.NewWithID("com.example.screenshotandexplain")

	// Load custom font
	fontData, err := ioutil.ReadFile("NotoSerifJP-VariableFont_wght.ttf")
	if err != nil {
		log.Fatal(err)
	}
	customFont := &fyne.StaticResource{
		StaticName:    "CustomFont",
		StaticContent: fontData,
	}

	// Set custom font theme
	a.Settings().SetTheme(&CustomFontTheme{font: customFont})

	w := a.NewWindow("Screenshot and Explain")

	textArea := widget.NewRichTextFromMarkdown("")
	textArea.Wrapping = fyne.TextWrapWord

	loadingSpinner := widget.NewProgressBarInfinite()
	loadingSpinner.Hide()

	btn := widget.NewButton("Take Screenshot and Explain", nil)
	btn.OnTapped = func() {
		textArea.ParseMarkdown("") // Clear the text area
		disableButtons(btn, loadingSpinner)
		go func() {
			response, err := takeScreenshotAndExplain()
			if err != nil {
				textArea.ParseMarkdown(fmt.Sprintf("Error: %v", err))
			} else {
				textArea.ParseMarkdown(response)
			}
			enableButtons(btn, loadingSpinner)
		}()
	}

	w.SetContent(container.NewVBox(btn, loadingSpinner, textArea))
	w.Resize(fyne.NewSize(800, 600)) // Set the window size to be larger
	w.ShowAndRun()
}

func disableButtons(btn *widget.Button, spinner *widget.ProgressBarInfinite) {
	btn.Disable()
	spinner.Show()
}

func enableButtons(btn *widget.Button, spinner *widget.ProgressBarInfinite) {
	btn.Enable()
	spinner.Hide()
}

func takeScreenshotAndExplain() (string, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	response, err := sendToChatGPT(imgBase64Str)
	if err != nil {
		return "", err
	}

	return response, nil
}

func sendToChatGPT(imageBase64 string) (string, error) {
	client := resty.New()
	apiKey := os.Getenv("API_KEY")
	gptModle := os.Getenv("GPT_MODEL")

	data := map[string]interface{}{
		"model": gptModle,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Explain the Japanese grammar in this image."},
					{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64," + imageBase64}},
				},
			},
		},
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey).
		SetBody(data).
		Post("https://api.openai.com/v1/chat/completions")

	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("unexpected response format")
	}

	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	return content, nil
}

// CustomFontTheme defines a theme that uses the custom font
type CustomFontTheme struct {
	font fyne.Resource
}

// Font returns the custom font for the given text style
func (c *CustomFontTheme) Font(s fyne.TextStyle) fyne.Resource {
	return c.font
}

// The following methods are required to implement the fyne.Theme interface
func (c *CustomFontTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (c *CustomFontTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (c *CustomFontTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
