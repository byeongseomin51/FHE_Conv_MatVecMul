package engine

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func ErrorPrint(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func txtToPlain(encoder *ckks.Encoder, txtPath string, params ckks.Parameters) *rlwe.Plaintext {

	file, err := os.Open(txtPath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		floats = append(floats, floatVal)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	//Make longer
	if len(floats) != 32768 {
		// fmt.Println(txtPath, " : Txt is short! 0 appended")
		for i := len(floats); i < 32768; i++ {
			floats = append(floats, 0)
		}
	}

	// encode to Plaintext
	exPlain := ckks.NewPlaintext(params, params.MaxLevel())
	err = encoder.Encode(floats, exPlain)
	if err != nil {
		fmt.Println(err)
	}

	return exPlain
}
func txtToFloat(txtPath string) []float64 {

	file, err := os.Open(txtPath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		floats = append(floats, floatVal)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	if len(floats) != 32768 {
		fmt.Println(txtPath, " : Txt is short! 0 appended")
		for i := len(floats); i < 32768; i++ {
			floats = append(floats, 0)
		}
	}

	return floats
}
func floatToPlain(floats []float64, encoder *ckks.Encoder, params ckks.Parameters) *rlwe.Plaintext {

	// encode to Plaintext
	exPlain := ckks.NewPlaintext(params, params.MaxLevel())
	encoder.Encode(floats, exPlain)

	return exPlain
}
func floatToTxt(filePath string, floats []float64) {

	if _, err := os.Stat(filePath); os.IsNotExist(err) {

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {

			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("File '%s' already exists. Overwrited\n", filePath)
	}

}
