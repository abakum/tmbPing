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
	fc := 0
	for i := 1; fc < 1; i++ {
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
			if i > f && c > 0 {
				fc++
			}
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

const lunarMonthDay float64 = 29.530588853
const gregorianEpoch = 1721425.5

func jDaP(t time.Time) (julianDays, agePart float64) {
	day_ := -2.0
	if leapGregorian(t.Year()) {
		day_ = -1.0
	}
	month_ := float64(t.Month()) + 1.0
	if month_ <= 2.0 {
		day_ = 0.0
	}
	day_ += float64(t.Day())
	year_ := float64(t.Year()) - 1.0
	julianDays = (gregorianEpoch - 1.0) +
		(365.0 * year_) +
		math.Floor(year_/4.0) +
		(-math.Floor(year_ / 100.0)) +
		math.Floor(year_/400.0) +
		math.Floor((((367.0*month_)-362.0)/12.0)+day_)
	agePart = normalize((julianDays - 2451550.1) / lunarMonthDay)
	return
}

func normalize(v float64) float64 {
	w := v - math.Floor(v)
	if w < 0 {
		return w + 1.0
	}
	return w
}

func moonPhase(t time.Time) string {
	//https://planetcalc.ru/524/
	//https://planetcalc.ru/personal/source/?id=522
	//https://gist.github.com/mrrrk/e100225508ad8b6882844de99d264ca7
	_, agePart := jDaP(t)
	ageDays := agePart * lunarMonthDay
	switch {
	case ageDays < 1.84566:
		return "🌑" //"NEW"
	case ageDays < 5.53699:
		return "🌒" //"Waxing crescent"
	case ageDays < 9.22831:
		return "🌓" //"First quarter"
	case ageDays < 12.91963:
		return "🌔" //"Waxing gibbous"
	case ageDays < 16.61096:
		return "🌕" //"FULL"
	case ageDays < 20.30228:
		return "🌖" //"Waning gibbous"
	case ageDays < 23.99361:
		return "🌗" //"Last quarter"
	case ageDays < 27.68493:
		return "🌘" //"Waning crescent"
	default:
		return "🌑" //"NEW"
	}
}

func moonZodiac(t time.Time) (zodiac string) {
	//https://web.archive.org/web/20090218203728/http://home.att.net/~srschmitt/lunarphasecalc.html Стефан Шмитт (Stephen R. Schmitt)
	julianDays, agePart := jDaP(t)
	IP2 := 4 * math.Pi * agePart
	DP := 2 * math.Pi * normalize((julianDays-2451562.2)/27.55454988)
	RP := normalize((julianDays - 2451555.8) / 27.321582241)
	LO := 360*RP + 6.3*math.Sin(DP) + 1.3*math.Sin(IP2-DP) + 0.7*math.Sin(IP2) //Moon's ecliptic longitude
	switch {
	case LO < 33.18:
		return "♓︎" //"Pisces"
	case LO < 51.16:
		return "♈︎" //"Aries"
	case LO < 93.44:
		return "♉︎" //"Taurus"
	case LO < 119.48:
		return "♊︎" //"Gemini"
	case LO < 135.30:
		return "♋︎" //"Cancer"
	case LO < 173.34:
		return "♌︎" //"Leo"
	case LO < 224.17:
		return "♍︎" //"Virgo"
	case LO < 242.57:
		return "♎︎" //"Libra"
	case LO < 271.26:
		return "♏︎" //"Scorpio"
	case LO < 302.49:
		return "♐︎" //"Sagittarius"
	case LO < 311.72:
		return "♑︎" //"Capricorn"
	case LO < 348.58:
		return "♒︎" //"Aquarius"
	default:
		return "♓︎" //"Pisces"
	}
}
