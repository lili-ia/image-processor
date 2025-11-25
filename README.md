# Image Processing Pipeline (Go)

This project demonstrates sequential and parallel (pipeline-based) image processing in Go.  
It loads JPG images from a directory, converts them to grayscale, applies a sepia filter, and saves the processed results.  
The program compares execution time between **sequential** and **parallel** processing.

---

## Requirements

You need:

- **Go 1.20+**
- A folder named `input_images/` containing `.jpg` files

---

## How to install and run

Clone the repository and run the project (all dependencies are already included in go.mod):

```bash
git clone https://github.com/lili-ia/image-processor
cd image-processor
go run .
```

## How to use

Place JPG images into the directory. Example
```bash
input_images/photo1.jpg
input_images/photo2.jpg
...
```
Run the application:
```bash
go run .
```
After processing completes, results will be stored in:
```
output_sequential/
output_parallel/
```
