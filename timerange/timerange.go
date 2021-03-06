package timerange

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const layoutYear = "2006"
const layoutMonth = "01"
const layoutDay = "02"

type dateFormat struct {
	missDay   bool
	missMonth bool
	missYear  bool
	fmt       string
}

var timeFormats = []dateFormat{
	{true, true, true, "15:04"},
	{true, true, true, "15:04:05"},
	{true, true, true, "15:04:05.00"},

	{false, false, false, "2006-01-02T15:04"},
	{false, false, false, "2006-01-02T15:04:05"},
	{false, false, false, "2006-01-02T15:04:05.00"},
	{false, false, false, "2006-01-02 15:04"},
	{false, false, false, "2006-01-02 15:04:05"},
	{false, false, false, "2006-01-02 15:04:05.00"},

	{false, true, true, "Mon 15:04"},
	{false, true, true, "Mon 15:04:05"},
	{false, true, true, "Mon 15:04:05.00"},

	{false, false, true, "Jan Mon 15:04"},
	{false, false, true, "Jan Mon 15:04:05"},
	{false, false, true, "Jan Mon 15:04:05.00"},
}

type TimeRange struct {
	From time.Time
	To   time.Time
}

func (tr TimeRange) String() string {
	format := "2006-01-02T15:04:05"
	return fmt.Sprintf("From='%s', To='%s'", tr.From.Format(format), tr.To.Format(format))
}

func ParseDate(str string) (time.Time, error) {
	var t time.Time
	var err error
	if str == "now" {
		return time.Now(), nil
	}

	r, _ := regexp.Compile("^-([0-9]+)([dhms])$")
	res := r.FindStringSubmatch(str)
	if len(res) != 0 {
		var subDuration time.Duration
		switch res[2] {
		case "d":
			subDuration = 24 * time.Hour
		case "h":
			subDuration = time.Hour
		case "m":
			subDuration = time.Minute
		case "s":
			subDuration = time.Second
		}

		multiplier, err := strconv.Atoi(res[1])
		if err != nil {
			panic(fmt.Sprintf("Failed to convert %s to int: %s", res[1], err.Error()))
		}

		subDuration = time.Duration(multiplier) * subDuration
		return time.Now().Add(-1 * subDuration), nil
	}

	for _, tf := range timeFormats {
		date := str
		layout := tf.fmt
		_, err = time.Parse(layout, date)
		if err != nil {
			continue
		}

		now := time.Now()

		// Layout "15:04" is treated like "0000-01-01 15:04:00"
		// golang's time is so weak and ugly
		// I could not find anything in golang as powefull as GNU date

		if tf.missYear {
			date = fmt.Sprintf("%d %s", now.Year(), date)
			layout = fmt.Sprintf("%s %s", layoutYear, layout)
		}

		if tf.missMonth {
			date = fmt.Sprintf("%02d %s", now.Month(), date)
			layout = fmt.Sprintf("%s %s", layoutMonth, layout)
		}

		if tf.missDay {
			date = fmt.Sprintf("%02d %s", now.Day(), date)
			layout = fmt.Sprintf("%s %s", layoutDay, layout)
		}

		t, err = time.Parse(layout, date)
		if err != nil {
			return t, fmt.Errorf("str=%s, layout=%s, %s", date, layout, err.Error())
		}

		return t, nil
	}

	return t, fmt.Errorf("no matching layout was found to parse %s", str)
}

func ParseRange(str string) (*TimeRange, error) {
	strs := strings.Split(str, "/")

	if len(strs) == 1 {
		str = fmt.Sprintf("%s/now", str)
		strs = strings.Split(str, "/")
	}

	if len(strs) != 2 {
		return nil, fmt.Errorf("%s splited into ambiguous amount of parts=%d (expected 2)", str, len(strs))
	}

	t := &TimeRange{}
	from, err := ParseDate(strs[0])
	if err != nil {
		return t, err
	}

	to, err := ParseDate(strs[1])
	if err != nil {
		return t, err
	}

	t.From = from
	t.To = to
	return t, nil
}
