package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NoF0rte/pdf"

	"github.com/PuerkitoBio/goquery"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// emailBillOrder 邮件订单
type emailBillOrder struct {
	Name   string `json:"name"`
	Time   string `json:"time"`
	Amount string `json:"amount"`
}

// fileBillOrder 出账单订单, 交易日,记账日,交易摘要,人民币金额,卡号末四位,交易地金额
type fileBillOrder struct {
	TradingDate   string `json:"tradingDate"`
	CreditDate    string `json:"creditDate"`
	Name          string `json:"name"`
	Amount        string `json:"amount"`
	TailNumber    string `json:"tailNumber"`
	TradingAmount string `json:"tradingAmount"`
}

var (
	// 07:53:41
	timeRe = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})`)
	// 07:53:41人民币 8.00尾号3885 消费 支付宝-xxxx有限公司
	orderRe    = regexp.MustCompile(`(?m)([\d:]+)人民币([  \-\d.,]+)(.*)`)
	orderCNYRe = regexp.MustCompile(`(?m)([\d:]+)CNY([  \-\d.,]+)(.*)`)
	// 2021-05-26
	date20210526 = time.Date(2021, 5, 26, 0, 0, 0, 0, time.Local)
	// 2021-10-31
	date20211031 = time.Date(2021, 10, 31, 0, 0, 0, 0, time.Local)
)

func emailParseCmb(cfg fetchConfig) ([]emailBillOrder, error) {
	var orders = make([]emailBillOrder, 0)
	c, err := client.DialTLS("imap.exmail.qq.com:993", nil)
	if err != nil {
		return orders, fmt.Errorf("dial imap server error: %s", err.Error())
	}
	defer func() {
		_ = c.Logout()
	}()

	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return orders, fmt.Errorf("email login error: %s", err.Error())
	}

	_, err = c.Select("inbox", true)
	if err != nil {
		return orders, fmt.Errorf("open inbox error: %s", err.Error())
	}

	filter := imap.NewSearchCriteria()
	filter.Since = time.Now().Add(time.Duration(-cfg.Hour) * time.Hour)
	seqNums, err := c.Search(filter)
	if err != nil {
		return orders, fmt.Errorf("search inbox error: %s", err.Error())
	}
	if len(seqNums) == 0 {
		log.Println("no new message")
		return orders, nil
	}

	seqSet := new(imap.SeqSet)
	for _, seqNum := range seqNums {
		seqSet.AddNum(seqNum)
	}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	section := &imap.BodySectionName{}
	go func() {
		done <- c.Fetch(seqSet, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			log.Println("Server didn't returned message body")
			continue
		}
		mr, err := mail.ReadMessage(r)
		if err != nil {
			log.Printf("read email error: %s\n", err.Error())
			continue
		}
		// "=?GBK?B?" + base64("每日信用管家", "GBK") + "?=" => =?gb2312?B?w7/I1dDF08O53LzS?=
		// 2023-06-23 之后邮件切换了编码
		// "=?utf-8?B?" + base64("每日信用管家", "utf-8") + "?=" => =?utf-8?B?5q+P5pel5L+h55So566h5a62?=
		if !strings.HasPrefix(mr.Header["Subject"][0], "=?gb2312?B?w7/I1dDF08O53LzS") && !strings.HasPrefix(mr.Header["Subject"][0], "=?utf-8?B?5q+P5pel5L+h55So566h5a6") {
			continue
		}
		// mail date
		// 2023-06-23 之后日期中添加了 (CST) => Fri, 23 Jun 2023 10:51:24 +0800 (CST)
		md, _ := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", strings.ReplaceAll(mr.Header.Get("Date"), " (CST)", ""))
		log.Printf("parsing mail @ %s\n", md.Format("2006-01-02"))
		// parse body
		mediaType, params, _ := mime.ParseMediaType(mr.Header.Get("Content-Type"))
		if strings.HasPrefix(mediaType, "multipart/") {
			br := multipart.NewReader(mr.Body, params["boundary"])
			for {
				p, err := br.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("email body parse error: %s\n", err.Error())
					break
				}

				slurp, err := io.ReadAll(p)
				if err != nil {
					log.Printf("email body parse error: %s\n", err.Error())
					break
				}
				if p.Header.Get("Content-Type") == "text/html; charset=\"gb2312\"" {
					html, err := base64.StdEncoding.DecodeString(string(slurp))
					if err != nil {
						log.Printf("decode body to html error: %s\n", err.Error())
						break
					}
					html, _ = simplifiedchinese.GBK.NewDecoder().Bytes(html)
					// parse html
					doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
					if err != nil {
						log.Printf("parse html error: %s\n", err.Error())
						break
					}
					text := strings.TrimSpace(doc.Text())
					if md.Before(date20210526) {
						orders = append(orders, parseBefore20210526(text)...)
					} else {
						orders = append(orders, parse(doc)...)
					}
				} else if p.Header.Get("Content-Type") == "text/html; charset=utf-8" {
					// 2023-06-23 之后邮件切换了编码 text/html; charset=utf-8
					// parse html
					doc, err := goquery.NewDocumentFromReader(bytes.NewReader(slurp))
					if err != nil {
						log.Printf("parse html error: %s\n", err.Error())
						break
					}
					orders = append(orders, parse(doc)...)
				}

			}
		}
	}

	if err := <-done; err != nil {
		return orders, err
	}
	return orders, nil
}

func fileParseCmb(file string) ([]fileBillOrder, error) {
	var lines = make([][]string, 0)
	f, r, err := pdf.Open(file)
	if err != nil {
		return nil, fmt.Errorf("fail to open file: %s", file)
	}
	defer func() {
		_ = f.Close()
	}()

	var year int
	var month int
	dateRe := regexp.MustCompile(`(\d{4})年(\d{2})月\d{2}日`)
	blankRe := regexp.MustCompile(`\s+`)
	// the x coordinate
	var x float64
	// the y coordinate
	var y float64
	var line string
	pages := r.NumPage()
	for i := 1; i <= pages; i++ {
		p := r.Page(i)
		texts := p.Content().Text
		// a text is just a word
		for _, text := range texts {
			// determine new line
			if math.Abs(text.Y-y) > 1 {
				// find year
				if year == 0 && dateRe.FindString(line) != "" {
					year, _ = strconv.Atoi(dateRe.FindAllStringSubmatch(line, -1)[0][1])
					month, _ = strconv.Atoi(dateRe.FindAllStringSubmatch(line, -1)[0][2])
				}
				// stop if reach end of file
				if strings.TrimSpace(line) == "本期账单金额" {
					break
				}
				if len(strings.Split(line, "|")) == 6 {
					// 1,100 to 1100
					line = strings.ReplaceAll(line, ",", "")
					// replace blank
					line = blankRe.ReplaceAllString(line, "")
					lines = append(lines, strings.Split(line, "|"))
				}
				x, y, line = 0, text.Y, ""
			}
			// determine split in one line
			if x != 0 && math.Abs(text.X-x) > 10 && len(strings.TrimSpace(line)) > 0 {
				line += "|"
			}
			x = text.X
			line += text.S
		}
	}
	// format data
	// determine if date cross year (transaction date increase in order)
	// skip title
	firstMon, err := strconv.Atoi(strings.Split(lines[1][0], "/")[0])
	if err != nil {
		firstMon, _ = strconv.Atoi(strings.Split(lines[2][0], "/")[0])
	}
	lastMon, _ := strconv.Atoi(strings.Split(lines[len(lines)-1][0], "/")[0])
	crossYear := (firstMon - lastMon) > 0

	// bill date
	billDate, _ := time.Parse("2006-01", fmt.Sprintf("%d-%02d", year, month))
	isBeforeDate20211031 := billDate.Before(date20211031)

	var orders []fileBillOrder

	var max int
	for _, l := range lines {
		// skip title
		if l[0] == "交易日" || l[0] == "SOLD" {
			continue
		}

		if isBeforeDate20211031 {
			// date format
			mon, _ := strconv.Atoi(strings.Split(l[0], "/")[0])
			if crossYear && mon >= max {
				max = mon
				l[0] = strconv.Itoa(year-1) + "-" + strings.ReplaceAll(l[0], "/", "-")
				l[4] = strconv.Itoa(year-1) + "-" + strings.ReplaceAll(l[4], "/", "-")
			} else {
				l[0] = strconv.Itoa(year) + "-" + strings.ReplaceAll(l[0], "/", "-")
				l[4] = strconv.Itoa(year) + "-" + strings.ReplaceAll(l[4], "/", "-")
			}

			orders = append(orders, fileBillOrder{
				TradingDate:   strings.ReplaceAll(l[0], " ", ""),
				Name:          strings.ReplaceAll(l[1], " ", ""),
				Amount:        l[2],
				TailNumber:    l[3],
				CreditDate:    strings.ReplaceAll(l[4], " ", ""),
				TradingAmount: l[5],
			})
		} else {
			// date format
			mon, _ := strconv.Atoi(strings.Split(l[0], "/")[0])
			if crossYear && mon >= max {
				max = mon
				l[0] = strconv.Itoa(year-1) + "-" + strings.ReplaceAll(l[0], "/", "-")
				l[1] = strconv.Itoa(year-1) + "-" + strings.ReplaceAll(l[1], "/", "-")
			} else {
				l[0] = strconv.Itoa(year) + "-" + strings.ReplaceAll(l[0], "/", "-")
				l[1] = strconv.Itoa(year) + "-" + strings.ReplaceAll(l[1], "/", "-")
			}

			orders = append(orders, fileBillOrder{
				TradingDate:   strings.ReplaceAll(l[0], " ", ""),
				CreditDate:    strings.ReplaceAll(l[1], " ", ""),
				Name:          strings.ReplaceAll(l[2], " ", ""),
				Amount:        l[3],
				TailNumber:    l[4],
				TradingAmount: l[5],
			})
		}
	}

	return orders, nil
}
