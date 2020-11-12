package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var (
	// 07:53:41
	timeRe = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})`)
	// 07:53:41人民币 8.00尾号3885 消费 支付宝-xxxx有限公司
	orderRe = regexp.MustCompile(`(?m)([\d:]+)人民币([  \d.,]+)(.*)`)
)

func cmb(cfg fetchConfig) ([]billOrder, error) {
	var orders = make([]billOrder, 0)
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
		// "=?GBK?B?" + base64("每日信用管家", "GBK") + "?="
		if !strings.HasPrefix(mr.Header["Subject"][0], "=?gb2312?B?w7/I1dDF08O53LzS") {
			continue
		}
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

				slurp, err := ioutil.ReadAll(p)
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
					var orderList []string
					idx := 0
					// 获取所有交易明细 -> 07:53:41人民币 8.00尾号3885 消费 支付宝-暖煨堂餐饮管理有限公司
					for i, match := range timeRe.FindAllString(text, -1) {
						if i == 0 {
							idx = strings.Index(text, match)
							continue
						}
						tmpIdx := strings.Index(text, match)
						orderList = append(orderList, strings.TrimSpace(text[idx:tmpIdx]))
						idx = tmpIdx
					}
					orderList = append(orderList, strings.TrimSpace(text[idx:strings.Index(text, "人民币消费")]))

					// 获取消费日期
					// 邮件内容变更兼容旧邮件
					dateStr := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
					if strings.Index(text, "截至") != -1 && strings.Index(text, "24时") != -1 {
						dateStr = strings.TrimSpace(text[strings.Index(text, "截至")+len("截至") : strings.Index(text, "24时")])
					} else if strings.Index(text, "消费人民币") != -1 {
						dateStr = strings.TrimSpace(text[strings.Index(text, "消费人民币")-12 : strings.Index(text, "消费人民币")-2])
					}
					dateStr = strings.ReplaceAll(dateStr, "/", "-")

					for _, order := range orderList {
						// 获取交易时间/金额/描述
						for _, match := range orderRe.FindAllStringSubmatch(order, -1) {
							timeStr := strings.TrimSpace(match[1])
							amountStr := strings.TrimSpace(strings.ReplaceAll(match[2], ",", ""))
							name := strings.TrimSpace(match[3])

							orders = append(orders, billOrder{Name: name, Time: dateStr + " " + timeStr, Amount: amountStr})
						}
					}
				}

			}
		}
	}

	if err := <-done; err != nil {
		return orders, err
	}
	return orders, nil
}
