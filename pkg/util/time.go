package util

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var zoneStr = time.Now().Format("-0700")

// 时间粒度常量
type TimeGranularity int

const (
	GranularityUnknown TimeGranularity = iota // 未知粒度
	GranularitySecond                         // 精确到秒
	GranularityMinute                         // 精确到分钟
	GranularityHour                           // 精确到小时
	GranularityDay                            // 精确到天
	GranularityMonth                          // 精确到月
	GranularityQuarter                        // 精确到季度
	GranularityYear                           // 精确到年
)

// timeOf 内部函数，解析各种格式的时间点，并返回时间粒度
// 支持以下格式:
// 1. 时间戳(秒): 1609459200 (GranularitySecond)
// 2. 标准日期: 20060102, 2006-01-02 (GranularityDay)
// 3. 带时间的日期: 20060102/15:04, 2006-01-02/15:04 (GranularityMinute)
// 4. 完整时间: 20060102150405 (GranularitySecond)
// 5. RFC3339: 2006-01-02T15:04:05Z07:00 (GranularitySecond)
// 6. 相对时间: 5h-ago, 3d-ago, 1w-ago, 1m-ago, 1y-ago (根据单位确定粒度)
// 7. 自然语言: now (GranularitySecond), today, yesterday (GranularityDay)
// 8. 年份: 2006 (GranularityYear)
// 9. 月份: 200601, 2006-01 (GranularityMonth)
// 10. 季度: 2006Q1, 2006Q2, 2006Q3, 2006Q4 (GranularityQuarter)
// 11. 年月日时分: 200601021504 (GranularityMinute)
func timeOf(str string) (t time.Time, g TimeGranularity, ok bool) {
	if str == "" {
		return time.Time{}, GranularityUnknown, false
	}

	str = strings.TrimSpace(str)

	// 处理自然语言时间
	switch strings.ToLower(str) {
	case "now":
		return time.Now(), GranularitySecond, true
	case "today":
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), GranularityDay, true
	case "yesterday":
		now := time.Now().AddDate(0, 0, -1)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), GranularityDay, true
	case "this-week":
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 { // 周日
			weekday = 7
		}
		// 本周一
		monday := now.AddDate(0, 0, -(weekday - 1))
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location()), GranularityDay, true
	case "last-week":
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 { // 周日
			weekday = 7
		}
		// 上周一
		lastMonday := now.AddDate(0, 0, -(weekday-1)-7)
		return time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, now.Location()), GranularityDay, true
	case "this-month":
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), GranularityMonth, true
	case "last-month":
		now := time.Now()
		return time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()), GranularityMonth, true
	case "this-year":
		now := time.Now()
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), GranularityYear, true
	case "last-year":
		now := time.Now()
		return time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location()), GranularityYear, true
	case "all":
		// 返回零值时间
		return time.Time{}, GranularityYear, true
	}

	// 处理相对时间: 5h-ago, 3d-ago, 1w-ago, 1m-ago, 1y-ago
	if strings.HasSuffix(str, "-ago") {
		str = strings.TrimSuffix(str, "-ago")

		// 特殊处理 0d-ago 为当天开始
		if str == "0d" {
			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), GranularityDay, true
		}

		// 解析数字和单位
		re := regexp.MustCompile(`^(\d+)([hdwmy])$`)
		matches := re.FindStringSubmatch(str)
		if len(matches) == 3 {
			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return time.Time{}, GranularityUnknown, false
			}

			// 确保数字是正数
			if num <= 0 {
				// 对于0d-ago已经特殊处理，其他0或负数都是无效的
				if num == 0 && matches[2] != "d" {
					return time.Time{}, GranularityUnknown, false
				}
				return time.Time{}, GranularityUnknown, false
			}

			now := time.Now()
			var resultTime time.Time
			var granularity TimeGranularity

			switch matches[2] {
			case "h": // 小时
				resultTime = now.Add(-time.Duration(num) * time.Hour)
				granularity = GranularityHour
			case "d": // 天
				resultTime = now.AddDate(0, 0, -num)
				granularity = GranularityDay
			case "w": // 周
				resultTime = now.AddDate(0, 0, -num*7)
				granularity = GranularityDay
			case "m": // 月
				resultTime = now.AddDate(0, -num, 0)
				granularity = GranularityMonth
			case "y": // 年
				resultTime = now.AddDate(-num, 0, 0)
				granularity = GranularityYear
			default:
				return time.Time{}, GranularityUnknown, false
			}

			return resultTime, granularity, true
		}

		// 尝试标准 duration 解析
		dur, err := time.ParseDuration(str)
		if err == nil {
			// 根据duration单位确定粒度
			hours := dur.Hours()
			if hours < 1 {
				return time.Now().Add(-dur), GranularitySecond, true
			} else if hours < 24 {
				return time.Now().Add(-dur), GranularityHour, true
			} else {
				return time.Now().Add(-dur), GranularityDay, true
			}
		}

		return time.Time{}, GranularityUnknown, false
	}

	// 处理季度: 2006Q1, 2006Q2, 2006Q3, 2006Q4
	if matched, _ := regexp.MatchString(`^\d{4}Q[1-4]$`, str); matched {
		re := regexp.MustCompile(`^(\d{4})Q([1-4])$`)
		matches := re.FindStringSubmatch(str)
		if len(matches) == 3 {
			year, _ := strconv.Atoi(matches[1])
			quarter, _ := strconv.Atoi(matches[2])

			// 验证年份范围
			if year < 1970 || year > 9999 {
				return time.Time{}, GranularityUnknown, false
			}

			// 计算季度的开始月份
			startMonth := time.Month((quarter-1)*3 + 1)

			return time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local), GranularityQuarter, true
		}
	}

	// 处理年份: 2006
	if len(str) == 4 && isDigitsOnly(str) {
		year, err := strconv.Atoi(str)
		if err == nil && year >= 1970 && year <= 9999 {
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local), GranularityYear, true
		}
		return time.Time{}, GranularityUnknown, false
	}

	// 处理月份: 200601 或 2006-01
	if (len(str) == 6 && isDigitsOnly(str)) || (len(str) == 7 && strings.Count(str, "-") == 1) {
		var year, month int
		var err error

		if len(str) == 6 && isDigitsOnly(str) {
			year, err = strconv.Atoi(str[0:4])
			if err != nil {
				return time.Time{}, GranularityUnknown, false
			}
			month, err = strconv.Atoi(str[4:6])
			if err != nil {
				return time.Time{}, GranularityUnknown, false
			}
		} else { // 2006-01
			parts := strings.Split(str, "-")
			if len(parts) != 2 {
				return time.Time{}, GranularityUnknown, false
			}
			year, err = strconv.Atoi(parts[0])
			if err != nil {
				return time.Time{}, GranularityUnknown, false
			}
			month, err = strconv.Atoi(parts[1])
			if err != nil {
				return time.Time{}, GranularityUnknown, false
			}
		}

		if year < 1970 || year > 9999 || month < 1 || month > 12 {
			return time.Time{}, GranularityUnknown, false
		}

		return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local), GranularityMonth, true
	}

	// 处理日期格式: 20060102 或 2006-01-02
	if len(str) == 8 && isDigitsOnly(str) {
		// 验证年月日
		year, _ := strconv.Atoi(str[0:4])
		month, _ := strconv.Atoi(str[4:6])
		day, _ := strconv.Atoi(str[6:8])

		if year < 1970 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, GranularityUnknown, false
		}

		// 进一步验证日期是否有效
		if !isValidDate(year, month, day) {
			return time.Time{}, GranularityUnknown, false
		}

		// 直接构造时间
		result := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		return result, GranularityDay, true
	} else if len(str) == 10 && strings.Count(str, "-") == 2 {
		// 验证年月日
		parts := strings.Split(str, "-")
		if len(parts) != 3 {
			return time.Time{}, GranularityUnknown, false
		}

		year, err1 := strconv.Atoi(parts[0])
		month, err2 := strconv.Atoi(parts[1])
		day, err3 := strconv.Atoi(parts[2])

		if err1 != nil || err2 != nil || err3 != nil {
			return time.Time{}, GranularityUnknown, false
		}

		if year < 1970 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, GranularityUnknown, false
		}

		// 进一步验证日期是否有效
		if !isValidDate(year, month, day) {
			return time.Time{}, GranularityUnknown, false
		}

		// 直接构造时间
		result := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		return result, GranularityDay, true
	}

	// 处理年月日时分: 200601021504
	if len(str) == 12 && isDigitsOnly(str) {
		year, _ := strconv.Atoi(str[0:4])
		month, _ := strconv.Atoi(str[4:6])
		day, _ := strconv.Atoi(str[6:8])
		hour, _ := strconv.Atoi(str[8:10])
		minute, _ := strconv.Atoi(str[10:12])

		if year < 1970 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 ||
			hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return time.Time{}, GranularityUnknown, false
		}

		// 进一步验证日期是否有效
		if !isValidDate(year, month, day) {
			return time.Time{}, GranularityUnknown, false
		}

		// 直接构造时间
		result := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
		return result, GranularityMinute, true
	}

	// 处理带时间的日期: 20060102/15:04 或 2006-01-02/15:04
	if strings.Contains(str, "/") {
		parts := strings.Split(str, "/")
		if len(parts) != 2 {
			return time.Time{}, GranularityUnknown, false
		}

		datePart := parts[0]
		timePart := parts[1]

		// 验证日期部分
		var year, month, day int
		var err1, err2, err3 error

		if len(datePart) == 8 && isDigitsOnly(datePart) {
			year, err1 = strconv.Atoi(datePart[0:4])
			month, err2 = strconv.Atoi(datePart[4:6])
			day, err3 = strconv.Atoi(datePart[6:8])
		} else if len(datePart) == 10 && strings.Count(datePart, "-") == 2 {
			dateParts := strings.Split(datePart, "-")
			if len(dateParts) != 3 {
				return time.Time{}, GranularityUnknown, false
			}
			year, err1 = strconv.Atoi(dateParts[0])
			month, err2 = strconv.Atoi(dateParts[1])
			day, err3 = strconv.Atoi(dateParts[2])
		} else {
			return time.Time{}, GranularityUnknown, false
		}

		if err1 != nil || err2 != nil || err3 != nil {
			return time.Time{}, GranularityUnknown, false
		}

		if year < 1970 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, GranularityUnknown, false
		}

		// 进一步验证日期是否有效
		if !isValidDate(year, month, day) {
			return time.Time{}, GranularityUnknown, false
		}

		// 验证时间部分
		if !regexp.MustCompile(`^\d{2}:\d{2}$`).MatchString(timePart) {
			return time.Time{}, GranularityUnknown, false
		}

		timeParts := strings.Split(timePart, ":")
		hour, err1 := strconv.Atoi(timeParts[0])
		minute, err2 := strconv.Atoi(timeParts[1])

		if err1 != nil || err2 != nil {
			return time.Time{}, GranularityUnknown, false
		}

		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return time.Time{}, GranularityUnknown, false
		}

		// 直接构造时间
		result := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
		return result, GranularityMinute, true
	}

	// 处理完整时间: 20060102150405
	if len(str) == 14 && isDigitsOnly(str) {
		year, _ := strconv.Atoi(str[0:4])
		month, _ := strconv.Atoi(str[4:6])
		day, _ := strconv.Atoi(str[6:8])
		hour, _ := strconv.Atoi(str[8:10])
		minute, _ := strconv.Atoi(str[10:12])
		second, _ := strconv.Atoi(str[12:14])

		if year < 1970 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 ||
			hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
			return time.Time{}, GranularityUnknown, false
		}

		// 进一步验证日期是否有效
		if !isValidDate(year, month, day) {
			return time.Time{}, GranularityUnknown, false
		}

		// 直接构造时间
		result := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		return result, GranularitySecond, true
	}

	// 处理时间戳(秒)
	if isDigitsOnly(str) {
		n, err := strconv.ParseInt(str, 10, 64)
		if err == nil {
			// 检查是否是合理的时间戳范围
			if n >= 1000000000 && n <= 253402300799 { // 2001年到2286年的秒级时间戳
				return time.Unix(n, 0), GranularitySecond, true
			}
		}
		return time.Time{}, GranularityUnknown, false
	}

	// 处理 RFC3339: 2006-01-02T15:04:05Z07:00
	if strings.Contains(str, "T") && (strings.Contains(str, "Z") || strings.Contains(str, "+") || strings.Contains(str, "-")) {
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			// 尝试不带秒的格式
			t, err = time.Parse("2006-01-02T15:04Z07:00", str)
		}
		if err == nil {
			return t, GranularitySecond, true
		}
	}

	// 排除所有其他不支持的格式
	return time.Time{}, GranularityUnknown, false
}

// TimeOf 解析各种格式的时间点
// 支持以下格式:
// 1. 时间戳(秒): 1609459200
// 2. 标准日期: 20060102, 2006-01-02
// 3. 带时间的日期: 20060102/15:04, 2006-01-02/15:04
// 4. 完整时间: 20060102150405
// 5. RFC3339: 2006-01-02T15:04:05Z07:00
// 6. 相对时间: 5h-ago, 3d-ago, 1w-ago, 1m-ago, 1y-ago (小时、天、周、月、年)
// 7. 自然语言: now, today, yesterday
// 8. 年份: 2006
// 9. 月份: 200601, 2006-01
// 10. 季度: 2006Q1, 2006Q2, 2006Q3, 2006Q4
// 11. 年月日时分: 200601021504
func TimeOf(str string) (t time.Time, ok bool) {
	t, _, ok = timeOf(str)
	return
}

// TimeRangeOf 解析各种格式的时间范围
// 支持以下格式:
// 1. 单个时间点: 根据时间粒度确定合适的时间范围
//   - 精确到秒/分钟/小时: 扩展为当天范围
//   - 精确到天: 当天 00:00:00 ~ 23:59:59
//   - 精确到月: 当月第一天 ~ 最后一天
//   - 精确到季度: 季度第一天 ~ 最后一天
//   - 精确到年: 当年第一天 ~ 最后一天
//
// 2. 时间区间: 2006-01-01~2006-01-31, 2006-01-01,2006-01-31, 2006-01-01 to 2006-01-31
// 3. 相对时间: last-7d, last-30d, last-3m, last-1y (最近7天、30天、3个月、1年)
// 4. 特定时间段: today, yesterday, this-week, last-week, this-month, last-month, this-year, last-year
// 5. all: 表示所有时间
func TimeRangeOf(str string) (start, end time.Time, ok bool) {
	if str == "" {
		return time.Time{}, time.Time{}, false
	}

	str = strings.TrimSpace(str)

	// 处理 all 特殊情况
	if strings.ToLower(str) == "all" {
		start = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
		return start, end, true
	}

	// 处理相对时间范围: last-7d, last-30d, last-3m, last-1y
	if matched, _ := regexp.MatchString(`^last-\d+[dwmy]$`, str); matched {
		re := regexp.MustCompile(`^last-(\d+)([dwmy])$`)
		matches := re.FindStringSubmatch(str)
		if len(matches) == 3 {
			num, err := strconv.Atoi(matches[1])
			if err != nil || num <= 0 {
				return time.Time{}, time.Time{}, false
			}

			now := time.Now()
			end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

			switch matches[2] {
			case "d": // 天
				start = now.AddDate(0, 0, -num)
				start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
				return start, end, true
			case "w": // 周
				start = now.AddDate(0, 0, -num*7)
				start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
				return start, end, true
			case "m": // 月
				start = now.AddDate(0, -num, 0)
				start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
				return start, end, true
			case "y": // 年
				start = now.AddDate(-num, 0, 0)
				start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
				return start, end, true
			}
		}
	}

	// 处理时间区间: 2006-01-01~2006-01-31, 2006-01-01,2006-01-31, 2006-01-01 to 2006-01-31
	separators := []string{"~", ",", " to "}
	for _, sep := range separators {
		if strings.Contains(str, sep) {
			parts := strings.Split(str, sep)
			if len(parts) == 2 {
				startTime, startGran, startOk := timeOf(strings.TrimSpace(parts[0]))
				endTime, endGran, endOk := timeOf(strings.TrimSpace(parts[1]))

				if startOk && endOk {
					// 根据粒度调整时间范围
					start = adjustStartTime(startTime, startGran)
					end = adjustEndTime(endTime, endGran)

					// 确保开始时间早于结束时间
					if start.After(end) {
						// 正确交换开始和结束时间
						start, end = adjustStartTime(endTime, endGran), adjustEndTime(startTime, startGran)
					}

					return start, end, true
				}
			}
		}
	}

	// 处理单个时间点，根据粒度确定合适的时间范围
	t, g, ok := timeOf(str)
	if ok {
		switch g {
		case GranularitySecond, GranularityMinute, GranularityHour:
			// 精确到秒/分钟/小时的时间点，扩展为当天范围
			start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			end = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		case GranularityDay:
			// 精确到天的时间点
			start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			end = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		case GranularityMonth:
			// 精确到月的时间点
			start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			end = time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 999999999, t.Location())
		case GranularityQuarter:
			// 精确到季度的时间点
			quarter := (t.Month()-1)/3 + 1
			startMonth := time.Month((int(quarter)-1)*3 + 1)
			endMonth := startMonth + 2
			start = time.Date(t.Year(), startMonth, 1, 0, 0, 0, 0, t.Location())
			end = time.Date(t.Year(), endMonth+1, 0, 23, 59, 59, 999999999, t.Location())
		case GranularityYear:
			// 精确到年的时间点
			start = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
			end = time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
		}
		return start, end, true
	}

	return time.Time{}, time.Time{}, false
}

// adjustStartTime 根据时间粒度调整开始时间
func adjustStartTime(t time.Time, g TimeGranularity) time.Time {
	switch g {
	case GranularitySecond, GranularityMinute, GranularityHour:
		// 对于精确到秒/分钟/小时的时间，保持原样
		return t
	case GranularityDay:
		// 精确到天，设置为当天开始
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case GranularityMonth:
		// 精确到月，设置为当月第一天
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case GranularityQuarter:
		// 精确到季度，设置为季度第一天
		quarter := (t.Month()-1)/3 + 1
		startMonth := time.Month((int(quarter)-1)*3 + 1)
		return time.Date(t.Year(), startMonth, 1, 0, 0, 0, 0, t.Location())
	case GranularityYear:
		// 精确到年，设置为当年第一天
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		// 未知粒度，默认为当天开始
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
}

// adjustEndTime 根据时间粒度调整结束时间
func adjustEndTime(t time.Time, g TimeGranularity) time.Time {
	switch g {
	case GranularitySecond, GranularityMinute, GranularityHour:
		// 对于精确到秒/分钟/小时的时间，保持原样
		return t
	case GranularityDay:
		// 精确到天，设置为当天结束
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
	case GranularityMonth:
		// 精确到月，设置为当月最后一天
		return time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 999999999, t.Location())
	case GranularityQuarter:
		// 精确到季度，设置为季度最后一天
		quarter := (t.Month()-1)/3 + 1
		startMonth := time.Month((int(quarter)-1)*3 + 1)
		endMonth := startMonth + 2
		return time.Date(t.Year(), endMonth+1, 0, 23, 59, 59, 999999999, t.Location())
	case GranularityYear:
		// 精确到年，设置为当年最后一天
		return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
	default:
		// 未知粒度，默认为当天结束
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
	}
}

// isDigitsOnly 检查字符串是否只包含数字
func isDigitsOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isValidDate 检查日期是否有效
func isValidDate(year, month, day int) bool {
	// 检查月份的天数
	daysInMonth := 31

	switch month {
	case 4, 6, 9, 11:
		daysInMonth = 30
	case 2:
		// 闰年判断
		if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
			daysInMonth = 29
		} else {
			daysInMonth = 28
		}
	}

	return day <= daysInMonth
}

func PerfectTimeFormat(start time.Time, end time.Time) string {
	endTime := end

	// 如果结束时间是某一天的 0 点整，将其减去 1 秒，视为前一天的结束
	if endTime.Hour() == 0 && endTime.Minute() == 0 && endTime.Second() == 0 && endTime.Nanosecond() == 0 {
		endTime = endTime.Add(-time.Second) // 减去 1 秒
	}

	// 判断是否跨年
	if start.Year() != endTime.Year() {
		return "2006-01-02 15:04:05" // 完整格式，包含年月日时分秒
	}

	// 判断是否跨天（但在同一年内）
	if start.YearDay() != endTime.YearDay() {
		return "01-02 15:04:05" // 月日时分秒格式
	}

	// 在同一天内
	return "15:04:05" // 只显示时分秒
}
