package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/geojson"
	"github.com/tidwall/geojson/geometry"
	"github.com/tidwall/gjson"
	"github.com/tidwall/rtree"
)

type hood struct {
	name string
	feat *geojson.Feature
}

type violation struct {
	point [2]float64
	row   int
	num   string
	hood  *hood
}

func main() {
	start := time.Now()

	mark := time.Now()
	fmt.Printf("Loading neighborhoods... ")
	hoods, err := loadHoods()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%.2f secs\n", time.Since(mark).Seconds())

	mark = time.Now()
	fmt.Printf("Loading violations... ")
	violations, err := loadViolations()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%.2f secs\n", time.Since(mark).Seconds())

	mark = time.Now()
	fmt.Printf("Joining neighborhoods and violations... ")
	join(hoods, violations)
	fmt.Printf("%.2f secs\n", time.Since(mark).Seconds())

	mark = time.Now()
	fmt.Printf("Writing output... ")
	if err := writeViolations(violations); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%.2f secs\n", time.Since(mark).Seconds())

	fmt.Printf("Total execution time... %.2f secs\n",
		time.Since(start).Seconds())
}

func loadHoods() (*rtree.RTree, error) {
	hoods := new(rtree.RTree)
	data, err := os.ReadFile("data/Neighborhoods_Philadelphia.json")
	if err != nil {
		return nil, err
	}
	json := string(data)
	g, err := geojson.Parse(json, nil)
	if err != nil {
		return nil, err
	}
	g.(*geojson.FeatureCollection).ForEach(func(f geojson.Object) bool {
		r := g.Rect()
		feat := f.(*geojson.Feature)
		h := &hood{
			name: gjson.Get(feat.Members(), "properties.LISTNAME").String(),
			feat: feat,
		}
		min, max := [2]float64{r.Min.X, r.Min.Y}, [2]float64{r.Max.X, r.Max.Y}
		hoods.Insert(min, max, h)
		return true
	})
	return hoods, nil
}

func loadViolations() ([]violation, error) {
	data, err := os.ReadFile("data/phl_parking.csv")
	if err != nil {
		return nil, err
	}
	csv := string(data)
	violations := make([]violation, 0, 10_000_000)
	var cols []string
	var row int
	s := strings.IndexByte(csv, '\n') + 1
	for i := s; i < len(csv); i++ {
		switch csv[i] {
		case ',':
			cols = append(cols, csv[s:i])
			s = i + 1
		case '\n':
			cols = append(cols, csv[s:i])
			var v violation
			v.point[0], _ = strconv.ParseFloat(string(cols[10]), 64)
			v.point[1], _ = strconv.ParseFloat(string(cols[9]), 64)
			v.num = cols[0]
			v.row = row
			violations = append(violations, v)
			s = i + 1
			cols = cols[:0]
			row++
		}
	}
	return violations, nil

}

func join(hoods *rtree.RTree, violations []violation) {
	var wg sync.WaitGroup
	ch := make(chan int, 8192)
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			for i := range ch {
				rpt := violations[i].point
				hoods.Search(rpt, rpt,
					func(_, _ [2]float64, v interface{}) bool {
						h := v.(*hood)
						gpt := geometry.Point{X: rpt[0], Y: rpt[1]}
						if h.feat.IntersectsPoint(gpt) {
							violations[i].hood = h
							return false
						}
						return true
					},
				)
			}
			wg.Done()
		}()
	}
	for i := range violations {
		ch <- i
	}
	close(ch)
	wg.Wait()
}

func writeViolations(violations []violation) error {
	f, err := os.Create("data/output.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = w.WriteString("anon_ticket_number,neighborhood\r\n")
	if err != nil {
		return err
	}
	var buf []byte
	var count int
	for _, v := range violations {
		if v.hood == nil {
			continue
		}
		buf = append(buf[:0], v.num...)
		buf = append(buf, ',')
		buf = append(buf, v.hood.name...)
		buf = append(buf, '\r', '\n')
		_, err = w.Write(buf)
		if err != nil {
			return err
		}
		count++
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}
