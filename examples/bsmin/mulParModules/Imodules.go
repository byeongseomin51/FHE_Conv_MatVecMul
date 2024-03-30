package mulParModules

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
func convertFloatToComplex(slice []float64) []complex128 {
	complexSlice := make([]complex128, len(slice))

	for i, v := range slice {
		complexSlice[i] = complex(v, 0)
	}

	return complexSlice
}
func txtToPlain(encoder *ckks.Encoder, txtPath string, params ckks.Parameters) *rlwe.Plaintext {
	// 파일 열기
	file, err := os.Open(txtPath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	// 파일 스캐너 생성
	scanner := bufio.NewScanner(file)

	// 각 줄 읽어오기
	for scanner.Scan() {
		// 문자열을 float64로 변환
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		// 슬라이스에 추가
		floats = append(floats, floatVal)
	}

	// 스캔 중 에러 확인
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	//Make longer
	if len(floats) != 32768 {
		fmt.Println(txtPath, " : Txt is short! 0 appended")
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
	// 파일 열기
	file, err := os.Open(txtPath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	// 파일 스캐너 생성
	scanner := bufio.NewScanner(file)

	// 각 줄 읽어오기
	for scanner.Scan() {
		// 문자열을 float64로 변환
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		// 슬라이스에 추가
		floats = append(floats, floatVal)
	}

	// 스캔 중 에러 확인
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	//Make longer
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
	// 파일이 이미 존재하는지 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("File '%s' already exists. Overwrited\n", filePath)
	}

}
