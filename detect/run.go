package main

import (
	"encoding/base64"
	"encoding/json"

	//"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"time"
	"unsafe"

	"github.com/Kagami/go-face"
)

// Path to directory with models and test images. Here it's assumed it
// points to the <https://github.com/Kagami/go-face-testdata> clone.
const dataDir = "go-face-testdata"

var (
	modelsDir = filepath.Join(dataDir, "models")
	imagesDir = filepath.Join(dataDir, "images")
)

type facesOrganizer struct {
	Faces           []faceElement
	Catagory2Person map[int32]string
}

func (faceOrg *facesOrganizer) AddFace(face faceElement) []faceElement {
	faceOrg.Faces = append(faceOrg.Faces, face)
	return faceOrg.Faces
}

type faceElement struct {
	Descriptor string
	Rectangle  image.Rectangle
	Catagory   int32
	Person     string
}

func main() {
	if os.Args[2] == "train" {
		train_images()
	}

	if os.Args[2] == "test" {
		rec := get_recognizer()
		test := load_data()
		ingest_samples(rec, test)
		catagorize_test(rec, test.Catagory2Person)
	}

	if os.Args[2] == "poll" {
		rec := get_recognizer()
		test := load_data()
		ingest_samples(rec, test)
		poll_file(rec, test.Catagory2Person)
	}

}

/* 	start := time.Now()
catID := rec.Classify(nayoungFace.Descriptor)
if catID < 0 {
	log.Fatalf("Can't classify")
}
// Code to measure
duration := time.Since(start)
// Formatted String
fmt.Printf("\nClassification Time: %v", duration)
fmt.Printf("\nSizeOf (bytes): %v\n", unsafe.Sizeof(rec)) */

func str2descr(s string) face.Descriptor {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return *(*face.Descriptor)(unsafe.Pointer(&b[0]))
}

func descr2str(d face.Descriptor) string {
	b := (*(*[unsafe.Sizeof(d)]byte)(unsafe.Pointer(&d)))[:]
	return base64.StdEncoding.EncodeToString(b)
}

func serialize_face(f face.Face, cat int32, person string) faceElement {
	dd := faceElement{
		descr2str(f.Descriptor),
		f.Rectangle,
		cat,
		person}
	log.Printf("Output Face %v", cat)
	return dd
}

func get_recognizer() face.Recognizer {
	rec, err := face.NewRecognizer(modelsDir)
	if err != nil {
		log.Fatalf("Error creating a recognizer")
	}

	return *rec
}

func face_recognize(rec face.Recognizer, image_path string) *face.Face {
	//log.Printf("Entering Face Recognizer: %v", rec)
	_face, err := rec.RecognizeSingleFile(image_path)
	//log.Printf("Exiting face Recognizer")
	if err != nil {
		log.Fatalf("Can't recognize: %v", err)
	}
	if _face == nil {
		log.Printf("No Faces: %v", image_path)
	}

	//log.Printf("Returning face")
	return _face
}

func load_data() facesOrganizer {
	inputFile := filepath.Join(dataDir, "models.json")
	file, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Error Unmarshalling Json Data")
	}
	var data facesOrganizer
	json.Unmarshal(file, &data)

	return data
}

func ingest_samples(rec face.Recognizer, faces facesOrganizer) {
	var samples []face.Descriptor
	var catagories []int32
	for i := 0; i < len(faces.Faces); i++ {
		samples = append(samples, str2descr(faces.Faces[i].Descriptor))
		catagories = append(catagories, faces.Faces[i].Catagory)
		//log.Printf("Catagory: %v, Person %v", faces.Faces[i].Catagory, faces.Faces[i].Person)
	}

	rec.SetSamples(samples, catagories)
}

func catagorize_test(rec face.Recognizer, people map[int32]string) {
	path := filepath.Join(imagesDir, "test")
	files, _ := ioutil.ReadDir(path)
	//log.Printf("Catagories: %v", people)

	for _, file := range files {
		file_path := filepath.Join(path, file.Name())
		log.Printf("FilePath: %v", file_path)
		_face := face_recognize(rec, file_path)
		catagory := rec.Classify(_face.Descriptor)

		//log.Printf("ID: %v", catagory)
		log.Printf("Found: %v", people[int32(catagory)])
	}
}

func poll_file(rec face.Recognizer, people map[int32]string) {
	path := filepath.Join("/tmp", "output.jpg")
	//log.Printf("Catagories: %v", people)
	var curr_person string

	for {

		_face := face_recognize(rec, path)
		catagory := rec.Classify(_face.Descriptor)

		if curr_person != people[int32(catagory)] {
			log.Printf("Found: %v", people[int32(catagory)])
			curr_person = people[int32(catagory)]
		}
		time.Sleep(1 * time.Second)
	}
}

func train_images() {
	var face_book facesOrganizer
	face_book.Catagory2Person = make(map[int32]string)

	rec := get_recognizer()
	dirs, _ := ioutil.ReadDir(imagesDir)

	for i, dir := range dirs {
		if dir.Name() != "test" {
			sub_dirs_path := filepath.Join(imagesDir, dir.Name())
			sub_dirs, _ := ioutil.ReadDir(sub_dirs_path)
			//Creates a catagory entry
			face_book.Catagory2Person[int32(i)] = dir.Name()

			for _, sub := range sub_dirs {
				file_path := filepath.Join(sub_dirs_path, sub.Name())
				log.Printf("Outputting for %v", file_path)

				_face := face_recognize(rec, file_path)
				if _face != nil {

					tmp_face := serialize_face(*_face, int32(i), dir.Name())
					face_book.AddFace(tmp_face)
				}
			}
		}
	}

	file, err := json.Marshal(face_book)
	if err != nil {
		log.Fatalf("Error Marshalling Json Data")
	}

	outputFile := filepath.Join(dataDir, "models.json")
	_ = ioutil.WriteFile(outputFile, file, 0644)
}

// trigger ffmpeg
/* func ffmpeg_process() {
	cmnd := exec.Command("ffmpeg", "-y", "-f", "v4l2", "-i", "/dev/video0", "-update", "1", "-r", "1", "/tmp/output.jpg")
	//cmnd.Run() // and wait
	cmnd.Start()
} */
