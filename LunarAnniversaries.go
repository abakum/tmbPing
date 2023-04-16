package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/abakum/MoonPhase"
	"github.com/starryalley/go-julianday"
)

func la(t time.Time) (years []string) {
	w0, wh := wd(t)
	m0 := MoonPhase.New(t)
	p0, ph := mPhase(m0)
	// p0, ph := tPhase(t)
	//z0, zh := moonZodiac(t)
	z0, zh := ZodiacSign(m0)
	f := time.Now().Year() - t.Year()
	years = append(years, fmt.Sprintf("#%s\n#%s\n#%s\n%d %s%s%s", wh, ph, zh, t.Year(), w0, p0, z0))
	fc := 0
	for i := 1; fc < 1; i++ {
		bd := t.AddDate(i, 0, 0)
		m := MoonPhase.New(bd)
		c := 0
		w, _ := wd(bd)
		if w0 == w {
			c++
		} else {
			w = "\u2003\u2004"
		}
		// p, _ := tPhase(bd)
		//p, _ := mPhase(m)
		p := m.PhaseNameLocale("")
		if p0 == p {
			c++
		} else {
			p = "\u2003\u2004"
		}
		//z, _ := moonZodiac(bd)
		z := m.ZodiacSignLocale("")
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

func wd(t time.Time) (string, string) {
	wds := map[int]string{
		1: "понедельник",
		2: "вторник",
		3: "среда",
		4: "четверг",
		5: "пятница",
		6: "суббота",
		7: "воскресенье",
	}
	iso := int(t.Weekday())
	if iso == 0 {
		iso = 7
	}
	return string([]rune{rune(int('0') + iso), '\uFE0F', '\u20E3'}), wds[iso]
}

const lunarMonthDay float64 = 29.530588853

func jDaP(t time.Time) (julianDays, agePart float64) {
	julianDays = julianday.Date(t)
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

func mPhase(m *MoonPhase.Moon) (string, string) {
	return m.PhaseNameLocale(""), strings.Replace(strings.ToLower(m.PhaseNameLocale("ru")), " ", "_", -1)
}

func ZodiacSign(m *MoonPhase.Moon) (string, string) {
	return m.ZodiacSignLocale(""), m.ZodiacSignLocale("ru")
}

func tPhase(t time.Time) (string, string) {
	//https://planetcalc.ru/524/
	//https://planetcalc.ru/personal/source/?id=522
	//https://gist.github.com/mrrrk/e100225508ad8b6882844de99d264ca7
	_, agePart := jDaP(t)
	ageDays := agePart * lunarMonthDay
	stdo.Println(t, ageDays)
	switch {
	case ageDays < 1.84566:
		return "🌑", "новолуние" //"NEW"
	case ageDays < 5.53699:
		return "🌒", "молодая_луна" //"Waxing crescent"
	case ageDays < 9.22831:
		return "🌓", "первая_четверть " //"First quarter"
	case ageDays < 12.91963:
		return "🌔", "прибывающая_луна" //"Waxing gibbous"
	case ageDays < 16.61096:
		return "🌕", "полнолуние" //"FULL"
	case ageDays < 20.30228:
		return "🌖", "убывающая_луна" //"Waning gibbous"
	case ageDays < 23.99361:
		return "🌗", "последняя_четверть" //"Last quarter"
	case ageDays < 27.68493:
		return "🌘", "старая_луна" //"Waning crescent"
	default:
		return "🌑", "новолуние" //"NEW"
	}
}

func moonZodiac(t time.Time) (string, string) {
	//https://web.archive.org/web/20090218203728/http://home.att.net/~srschmitt/lunarphasecalc.html Стефан Шмитт (Stephen R. Schmitt)
	// temp = (jgDay - 2451555.8) / 27.321582241;
	// float rp = this.normIt(temp);
	// temp  = (360.0 * rp) + (6.3*sin(dp));
	// temp += 1.3*sin(2.0*ipRad - dp);
	// temp += 0.7*sin(2.0*ipRad);
	// this.eclipticLongitude = temp;

	julianDays, agePart := jDaP(t)
	IP2 := 4 * math.Pi * agePart
	DP := 2 * math.Pi * normalize((julianDays-2451562.2)/27.55454988)
	RP := normalize((julianDays - 2451555.8) / 27.321582241)
	LO := 360*RP + 6.3*math.Sin(DP) + 1.3*math.Sin(IP2-DP) + 0.7*math.Sin(IP2) //Moon's ecliptic longitude
	stdo.Println(julianDays, agePart, IP2, DP, RP, LO)
	switch {
	case LO < 33.18:
		return "♈︎", "овен" //"Aries"
	case LO < 51.16:
		return "♉︎", "телец" //"Taurus"
	case LO < 93.44:
		return "♊︎", "близнецы" //"Gemini"
	case LO < 119.48:
		return "♋︎", "рак" //"Cancer"
	case LO < 135.30:
		return "♌︎", "лев" //"Leo"
	case LO < 173.34:
		return "♍︎", "дева" //"Virgo"
	case LO < 224.17:
		return "♎︎", "весы" //"Libra"
	case LO < 242.57:
		return "♏︎", "скорпион" //"Scorpio"
	case LO < 271.26:
		return "♐︎", "стрелец" //"Sagittarius"
	case LO < 302.49:
		return "♑︎", "козерог" //"Capricorn"
	case LO < 311.72:
		return "♒︎", "водолей" //"Aquarius"
	case LO < 348.58:
		return "♓︎", "рыбы" //"Pisces"
	default:
		return "♈︎", "овен" //"Aries"
	}
}
