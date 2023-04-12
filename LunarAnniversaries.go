package main

import (
	"fmt"
	"math"
	"time"
)

func la(t time.Time) (years []string) {
	years = make([]string, 0)
	w0 := wd(t)
	p0 := moonPhase(t)
	z0 := moonZodiac(t)
	f := time.Now().Year() - t.Year()
	years = append(years, fmt.Sprintf("%d %s%s%s", t.Year(), w0, p0, z0))
	for i := 1; len(years) < 12; i++ {
		bd := t.AddDate(i, 0, 0)
		c := 0
		w := wd(bd)
		if w0 == w {
			c++
		} else {
			w = "\u2003\u2004"
		}
		p := moonPhase(bd)
		if p0 == p {
			c++
		} else {
			p = "\u2003\u2004"
		}
		z := moonZodiac(bd)
		if z0 == z {
			c++
		} else {
			z = ""
		}
		if i > f && c > 0 || c > 1 {
			years = append(years, fmt.Sprintf("%d %s%s%s", bd.Year(), w, p, z))
		}
	}
	return
}

func wd(bd time.Time) string {
	return string([]rune{rune(int('0') + int(bd.Weekday())), '\uFE0F', '\u20E3'})
}

func leapGregorian(year int) bool {
	return ((year % 4) == 0) &&
		(!(((year % 100) == 0) && ((year % 400) != 0)))
}

func gregorianToJd(year, month, day float64) float64 {
	l := -2.0
	if leapGregorian(int(year)) {
		l = -1.0
	}
	if month <= 2.0 {
		l = 0.0
	}
	return (1721425.5 - 1.0) +
		(365.0 * (year - 1.0)) +
		math.Floor((year-1.0)/4.0) +
		(-math.Floor((year - 1.0) / 100.0)) +
		math.Floor((year-1.0)/400.0) +
		math.Floor((((367.0*month)-362.0)/12.0)+
			l+day)
}
func normalize(v float64) float64 {
	w := v - math.Floor(v)
	if w < 0 {
		return w + 1.0
	}
	return w
}

func moonPhase(grig time.Time) string {
	//https://gist.github.com/mrrrk/e100225508ad8b6882844de99d264ca7
	ageDays := normalize((gregorianToJd(float64(grig.Year()), float64(grig.Month()+1.0), float64(grig.Day()))-2451550.1)/lunarMonthDay) * lunarMonthDay
	switch {
	case ageDays < 1.84566:
		return "🌑"
	case ageDays < 5.53699:
		return "🌒"
	case ageDays < 9.22831:
		return "🌓"
	case ageDays < 12.91963:
		return "🌔"
	case ageDays < 16.61096:
		return "🌕"
	case ageDays < 20.30228:
		return "🌖"
	case ageDays < 23.99361:
		return "🌗"
	case ageDays < 27.68493:
		return "🌘"
	default:
		return "🌑"
	}
}

const lunarMonthDay float64 = 29.530588853

func moonZodiac(grig time.Time) (zodiac string) {
	julianDays := gregorianToJd(float64(grig.Year()), float64(grig.Month()+1.0), float64(grig.Day()))
	IP := 4 * math.Pi * normalize((julianDays-2451550.1)/lunarMonthDay)
	DP := 2 * math.Pi * normalize((julianDays-2451562.2)/27.55454988)
	RP := normalize((julianDays - 2451555.8) / 27.321582241)
	LO := 360*RP + 6.3*math.Sin(DP) + 1.3*math.Sin(IP-DP) + 0.7*math.Sin(IP)
	switch {
	case LO < 33.18:
		return "♓︎"
	case LO < 51.16:
		return "♈︎"
	case LO < 93.44:
		return "♉︎"
	case LO < 119.48:
		return "♊︎"
	case LO < 135.30:
		return "♋︎"
	case LO < 173.34:
		return "♌︎"
	case LO < 224.17:
		return "♍︎"
	case LO < 242.57:
		return "♎︎"
	case LO < 271.26:
		return "♏︎"
	case LO < 302.49:
		return "♐︎"
	case LO < 311.72:
		return "♑︎"
	case LO < 348.58:
		return "♒︎"
	default:
		return "♓︎"
	}
}
