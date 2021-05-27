package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/frizinak/subscene/fuzzy"
	"github.com/frizinak/subscene/subscene"
	"github.com/mattn/go-runewidth"
)

var fileQueryRE = regexp.MustCompile(`1080p|720p|1080|720|4k`)

func intRange(a string) ([]int, bool) {
	comma := strings.Split(a, ",")
	r := make([]int, 0, len(comma))
	for _, n := range comma {
		dash := strings.SplitN(n, "-", 2)
		v, err := strconv.Atoi(strings.TrimSpace(dash[0]))
		if err != nil {
			return r, false
		}
		if len(dash) != 2 {
			r = append(r, v)
			continue
		}

		if len(dash) == 2 {
			v2, err := strconv.Atoi(strings.TrimSpace(dash[1]))
			if err != nil {
				return r, false
			}
			if v2 < v {
				return r, false
			}
			for i := v; i <= v2; i++ {
				r = append(r, i)
			}
		}
	}

	return r, len(r) > 0
}

func choice(list []string, w int) ([]int, error) {
	n := 2
	dp := 1
	for ln := len(list); ln >= 10; ln /= 10 {
		dp++
	}

	f := "\033[1;34m %0" + strconv.Itoa(dp) + "d \033[0m\033[31m%s\033[0m\n"
	for i, r := range list {
		r = runewidth.FillRight(runewidth.Truncate(r, w-n-dp, "..."), w-n-dp)
		fmt.Printf(f, i+1, r)
	}

	sc := bufio.NewScanner(os.Stdin)
	sc.Split(bufio.ScanLines)

	var ints []int
	var ok bool
	for {
		fmt.Print("\033[34mWhich? \033[0m")
		if !sc.Scan() {
			break
		}

		ints, ok = intRange(strings.TrimSpace(sc.Text()))
		if ok && len(ints) != 0 {
			for i, choice := range ints {
				choice--
				ints[i] = choice
				if choice < 0 && choice <= len(list) {
					ok = false
					break
				}
			}
			if ok {
				break
			}
		}
	}

	fmt.Print("\033[0m")
	if err := sc.Err(); err != nil {
		panic(err)
	}
	if len(ints) == 0 {
		return nil, errors.New("stdin closed unexpectedly")
	}

	return ints, nil
}

func exit(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func termSize() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	res, _ := cmd.Output()
	out := strings.Fields(strings.TrimSpace(string(res)))
	if len(out) != 2 {
		return 0, 0
	}
	x, _ := strconv.Atoi(out[1])
	y, _ := strconv.Atoi(out[0])

	return x, y
}

func main() {
	var i bool
	var q bool
	var lang string
	var hi bool
	flag.StringVar(&lang, "l", string(subscene.LangEnglish), "subtitle language")
	flag.BoolVar(&i, "i", false, "run interactively instead of picking the first result")
	flag.BoolVar(&hi, "hi", false, "prefer subtitles for the hearing impaired")
	flag.BoolVar(&q, "q", false, "sush")
	flag.Usage = func() {
		fmt.Println("Usage of subscene")
		fmt.Println("subscene [opts] <media query> <subtitle query>")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("<title query>:")
		fmt.Println("    The media title to query subscene.com for.")
		fmt.Println()
		fmt.Println("<subtitle query>:")
		fmt.Println("    The subtitle query we will try to find the best fuzzy match for.")
		fmt.Println("    If this path exists and")
		fmt.Println("         is a directory: the directory name will be used as query")
		fmt.Println("                         and the subtitles will be unzipped here.")
		fmt.Println("         is a file:      the filename without extension will be used as query")
		fmt.Println("                         and only the first subtitle will be stored with the same")
		fmt.Println("                         filename + '.srt'.")
		fmt.Println("                         e.g.: subscene 'line of duty second' ~/owneddvdrips/line-of-duty-s02e03.avi")
		fmt.Println("                               should result in ~/owneddvdrips/line-of-duty-s02e03.srt")
		fmt.Println()
	}
	flag.Parse()

	w, _ := termSize()

	query := strings.TrimSpace(flag.Arg(0))
	if query == "" {
		exit(errors.New("please provide a query"))
	}

	path := filepath.Clean(flag.Arg(1))
	_path, err := filepath.Abs(path)
	if err == nil {
		path = _path
	}

	dir := "./"
	fq := filepath.Base(path)
	ext := filepath.Ext(fq)
	fq = fq[:len(fq)-len(ext)]
	fp := ""
	if stat, _ := os.Stat(path); stat != nil {
		fp = fq
		dir = filepath.Dir(path)
		if stat.IsDir() {
			fp = ""
			dir = path
		}
	}
	fq = fileQueryRE.ReplaceAllString(fq, "")

	api := subscene.New(nil)

	res, err := api.Search(query, 30)
	exit(err)

	if len(res) == 0 {
		exit(errors.New("no results"))
	}

	strs := make(sort.StringSlice, len(res))
	for i, r := range res {
		strs[i] = r.String()
	}

	fuzzy.Search(query, strs, strs, res)

	ixs := []int{0}
	if i {
		for i := range strs {
			strs[i] = fmt.Sprintf("%s", strs[i])
		}
		ixs, err = choice(strs, w)
		fmt.Println()
		exit(err)
	}

	if !q {
		for _, ix := range ixs {
			s := runewidth.FillRight(res[ix].Title, w-2)
			fmt.Printf("\033[30;43m  %-s\033[0m\n", s)
		}
		fmt.Println()
	}

	downloads := make(subscene.Downloads, 0)
	for _, ix := range ixs {
		dls, err := api.Subtitles(res[ix], 100)
		exit(err)
		downloads = append(downloads, dls...)
	}
	strs = strs[:0]
	downloads = downloads.FilterLanguage(subscene.Language(lang))
	for _, dl := range downloads {
		hi := ""
		if dl.HI {
			hi = "HI"
		}
		strs = append(strs, fmt.Sprintf("%2s %-60s", hi, dl.Title))
	}

	if fq != "" {
		fuzzy.Search(fq, strs, strs, downloads)
		top := regexp.MustCompile(
			`(?i)` +
				`s[0-9]{2,}e[0-9]{2,}|` + // S02E04
				`season [0-9]+|` +
				`(?:19|20)[0-9]{2}`, // 1900-2099
		)

		ms := top.FindAllString(fq, -1)
		headstr := make([]string, 0, len(strs))
		headdl := make(subscene.Downloads, 0, len(downloads))
		for _, m := range ms {
			m = strings.ToLower(m)
			for i := 0; i < len(strs); i++ {
				if strings.Contains(strings.ToLower(strs[i]), m) {
					headstr = append(headstr, strs[i])
					headdl = append(headdl, downloads[i])
					strs = append(strs[:i], strs[i+1:]...)
					downloads = append(downloads[:i], downloads[i+1:]...)
					i--
				}
			}
		}

		if len(headstr) != 0 {
			strs = append(headstr, strs...)
			downloads = append(headdl, downloads...)
		}
	}

	for i, m := 0, 0; i < len(strs)-m; i++ {
		if downloads[i].HI != hi {
			curstr, curdl := strs[i], downloads[i]
			for j := i; j < len(strs)-1; j++ {
				strs[j] = strs[j+1]
				downloads[j] = downloads[j+1]
			}
			strs[len(strs)-1] = curstr
			downloads[len(downloads)-1] = curdl
			m++
			i--
		}
	}

	if i {
		ixs, err = choice(strs, w)
		fmt.Println()
		exit(err)
	}

	downloads = downloads.FilterN(ixs...)
	if len(downloads) == 0 {
		exit(errors.New("no results"))
	}

	if !q {
		for _, dl := range downloads {
			s := runewidth.FillRight(dl.Title, w-2)
			fmt.Printf("\033[30;43m  %s\033[0m\n", s)
		}
		fmt.Println()
	}

	cb := func(i subscene.ZipInfo) {
		if q {
			return
		}

		if i.Err != nil {
			fmt.Printf(
				"\033[1;37;41m Fail \033[0m %s\n%s\n%s\n\n",
				i.Err,
				i.URI.String(),
				i.Filename,
			)
			return
		}

		fmt.Printf("\033[1;30;42m Downloaded \033[0m %s\n", i.Filename)
		for k, v := range i.Extracted {
			if v == "" {
				v = "skipped"
			}
			fmt.Printf("    - %s -> %s\n", k, v)
		}
		fmt.Println()
	}

	exit(api.Get(downloads, dir, fp, 20, cb))
	exit(err)
	if !q {
		fmt.Println("Done")
	}
}
