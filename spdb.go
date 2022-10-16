package main

import (
	"fmt"
	"github.com/NoF0rte/pdf"
	"os"
	"regexp"
	"strings"
	"time"
)

func openEncryptedPdf(file string, password string) (*os.File, *pdf.Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	retry := 0
	reader, err := pdf.NewReaderEncrypted(f, fi.Size(), func() string {
		if retry > 0 {
			return ""
		}
		retry += 1
		return password
	})
	return f, reader, err
}

func fileParseSpdb(file string, password string) ([]fileBillOrder, error) {
	var lines = make([][]string, 0)
	f, r, err := openEncryptedPdf(file, password)
	if err != nil {
		return nil, fmt.Errorf("fail to open file: %s", file)
	}
	defer func() {
		_ = f.Close()
	}()

	dateRe := regexp.MustCompile(`^2\d{7}$`)
	pageFootRe := regexp.MustCompile(`^第\d*页/共\d*页，Page \d* of \d*$`)
	// the x coordinate
	var x float64
	var part string
	var line string
	findContent := false
	pages := r.NumPage()
	for i := 1; i <= pages; i++ {
		p := r.Page(i)
		texts := p.Content().Text
		// a text is just a word
		for _, text := range texts {
			// fmt.Printf("%d %.f,%.f %s\n", len(lines), text.X, text.Y, text.S)
			// determine new line
			if text.X != x {
				// 如果遇到新的交易日期、新一行
				if findContent && dateRe.Match([]byte(part)) {
					if line != "" {
						lines = append(lines, strings.Split(strings.TrimSuffix(line, "|"), "|"))
					}
					line = ""
				}
				// 页脚, eg: 第1页/共2页，Page 1 of 2
				if pageFootRe.Match([]byte(part)) {
					x, part = text.X, text.S
					continue
				}
				if findContent {
					line += part + "|"
				}

				// "Summary" 以下为内容
				if part == "Summary" {
					findContent = true
				}
				x, part = 0, ""
			}
			x = text.X
			part += text.S
		}
	}

	// format data
	var orders []fileBillOrder
	for _, l := range lines {
		if len(l) < 6 {
			return nil, fmt.Errorf("not support data size less than 6")
		}
		td, _ := time.Parse("20060102", l[0])
		cardNumber := l[2]
		var order = fileBillOrder{
			TradingDate:   td.Format("2006-01-02"),
			Name:          l[3],
			Amount:        l[4],
			TailNumber:    cardNumber[len(cardNumber)-4:],
			CreditDate:    td.Format("2006-01-02"),
			TradingAmount: l[4],
		}
		if len(l) > 6 {
			order.Name = l[6]
		}
		orders = append(orders, order)
	}
	return orders, nil
}
