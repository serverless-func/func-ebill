package main

import (
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
	"time"
)

func parse(doc *goquery.Document) []emailBillOrder {
	var orders = make([]emailBillOrder, 0)
	var orderList []string
	dateStr := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	doc.Find("#fixBand3 > table > tbody > tr").Each(func(i int, s *goquery.Selection) {
		text := regexp.MustCompile(`\s+`).ReplaceAllString(s.Text(), "")
		if i > 0 {
			// 11:26:34CNY 12.00尾号3885 消费 支付宝-南京天使谷酒店管理有限公司
			orderList = append(orderList, text)
		} else {
			dateStr = strings.TrimSpace(text[:strings.Index(text, "您的消费")])
		}
	})
	dateStr = strings.ReplaceAll(dateStr, "/", "-")
	for _, order := range orderList {
		// 获取交易时间/金额/描述
		for _, match := range orderCNYRe.FindAllStringSubmatch(order, -1) {
			timeStr := strings.TrimSpace(match[1])
			amountStr := strings.TrimSpace(strings.ReplaceAll(match[2], ",", ""))
			name := strings.TrimSpace(match[3])

			orders = append(orders, emailBillOrder{Name: name, Time: dateStr + " " + timeStr, Amount: amountStr})
		}
	}
	return orders
}

func parseBefore20210526(text string) []emailBillOrder {
	var orders = make([]emailBillOrder, 0)
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

			orders = append(orders, emailBillOrder{Name: name, Time: dateStr + " " + timeStr, Amount: amountStr})
		}
	}
	return orders
}
