package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type ImageTask struct {
	FilePath string
	Img      image.Image
}

func toGrayscale(img image.Image) image.Image {
	bounds := img.Bounds()
	grayImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			gray := uint8((0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256)
			grayImg.SetRGBA(x, y, color.RGBA{R: gray, G: gray, B: gray, A: uint8(a / 256)})
		}
	}
	return grayImg
}

func toSepia(img image.Image) image.Image {
	bounds := img.Bounds()
	sepiaImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA).RGBA()

			r8, g8, b8 := uint8(r/257), uint8(g/257), uint8(b/257)

			newR := float64(r8)*0.393 + float64(g8)*0.769 + float64(b8)*0.189
			newG := float64(r8)*0.349 + float64(g8)*0.686 + float64(b8)*0.168
			newB := float64(r8)*0.272 + float64(g8)*0.534 + float64(b8)*0.131

			sepiaImg.SetRGBA(x, y, color.RGBA{
				R: uint8(min(255, int(newR))),
				G: uint8(min(255, int(newG))),
				B: uint8(min(255, int(newB))),
				A: uint8(a / 256),
			})
		}
	}
	return sepiaImg
}

func loadWorker(filePaths <-chan string, tasksChan chan<- ImageTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			log.Printf("Error opening file %s: %v", path, err)
			continue
		}
		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			log.Printf("Error decoding image %s: %v", path, err)
			continue
		}
		tasksChan <- ImageTask{FilePath: path, Img: img}
	}
}

func processWorker(tasksChan <-chan ImageTask, resultsChan chan<- ImageTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasksChan {
		grayImg := toGrayscale(task.Img)
		sepiaImg := toSepia(grayImg)

		task.Img = sepiaImg
		resultsChan <- task
	}
}

func saveWorker(resultsChan <-chan ImageTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range resultsChan {
		outputPath := filepath.Join("output_parallel", filepath.Base(task.FilePath))

		outFile, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Error creating output file %s: %v", outputPath, err)
			continue
		}
		defer outFile.Close()

		if err := jpeg.Encode(outFile, task.Img, nil); err != nil {
			log.Printf("Error encoding JPEG %s: %v", outputPath, err)
		}
	}
}

func runParallelPipeline(filePaths []string, numWorkers int) {
	fmt.Printf("--- Паралельний Режим (Workers: %d) ---\n", numWorkers)
	start := time.Now()

	filesChan := make(chan string, len(filePaths))
	tasksChan := make(chan ImageTask, numWorkers)
	resultsChan := make(chan ImageTask, numWorkers)

	var wgLoad sync.WaitGroup
	var wgProcess sync.WaitGroup
	var wgSave sync.WaitGroup

	os.MkdirAll("output_parallel", fs.ModePerm)

	wgSave.Add(1)
	go func() {
		saveWorker(resultsChan, &wgSave)
	}()

	for i := 0; i < numWorkers; i++ {
		wgProcess.Add(1)
		go processWorker(tasksChan, resultsChan, &wgProcess)
	}

	wgLoad.Add(1)
	go func() {
		loadWorker(filesChan, tasksChan, &wgLoad)
	}()

	for _, path := range filePaths {
		filesChan <- path
	}
	close(filesChan)

	wgLoad.Wait()
	close(tasksChan)

	wgProcess.Wait()
	close(resultsChan)

	wgSave.Wait()

	duration := time.Since(start)
	fmt.Printf("Час виконання (Паралельний): %s\n", duration)
}

func runSequential(filePaths []string) {
	fmt.Println("--- Послідовний Режим ---")
	start := time.Now()

	os.MkdirAll("output_sequential", fs.ModePerm)

	for _, path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			log.Printf("Error opening file %s: %v", path, err)
			continue
		}
		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			log.Printf("Error decoding image %s: %v", path, err)
			continue
		}

		grayImg := toGrayscale(img)
		sepiaImg := toSepia(grayImg)

		outputPath := filepath.Join("output_sequential", filepath.Base(path))
		outFile, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Error creating output file %s: %v", outputPath, err)
			continue
		}
		jpeg.Encode(outFile, sepiaImg, nil)
		outFile.Close()
	}

	duration := time.Since(start)
	fmt.Printf("Час виконання (Послідовний): %s\n", duration)
}

func main() {
	log.SetFlags(0)

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	fmt.Printf("Використовується ядер CPU: %d\n", numCPU)

	testDir := "input_images"
	os.MkdirAll(testDir, fs.ModePerm)

	filePaths, err := filepath.Glob(filepath.Join(testDir, "*.jpg"))
	if err != nil {
		log.Fatal(err)
	}

	if len(filePaths) == 0 {
		fmt.Println("У директорії 'input_images' відсутні зображення.")
	}

	fmt.Printf("Знайдено файлів для обробки: %d\n", len(filePaths))
	fmt.Println("--------------------------------------------------")

	runSequential(filePaths)

	fmt.Println("--------------------------------------------------")

	runParallelPipeline(filePaths, numCPU)

	fmt.Println("--------------------------------------------------")
	fmt.Println("Обробку успішно завершено.")
}
