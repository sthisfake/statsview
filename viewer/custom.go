package viewer

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var chunkOfData [][]string

const (
	// VGCSzie is the name of GCSizeViewer
	custom = "custom"
)

// GCSizeViewer collects the GC size metric via `runtime.ReadMemStats()`
type CustomViewer struct {
	smgr  *StatsMgr
	graph *charts.Line
}

// NewGCSizeViewer returns the GCSizeViewer instance
// Series: GCSys / NextGC
func NewCustom() Viewer {
	graph := newBasicView(VGCSize)
	graph.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Custom"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Size", AxisLabel: &opts.AxisLabel{Formatter: "{value} MB"}}),
	)
	graph.AddSeries("GCSys", []opts.LineData{}).
		AddSeries("NextGC", []opts.LineData{})

	return &GCSizeViewer{graph: graph}
}

func (vr *CustomViewer) SetStatsMgr(smgr *StatsMgr) {
	vr.smgr = smgr
}

func (vr *CustomViewer) Name() string {
	return custom
}

func (vr *CustomViewer) View() *charts.Line {
	return vr.graph
}

func (vr *CustomViewer) Serve(w http.ResponseWriter, _ *http.Request) {
	conn, err := net.Dial("tcp", "localhost:5000")
	if err != nil {
		log.Fatal("Error connecting to server:", err)
	}
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for {
		chunkOfData = make([][]string, 4)
		for i := 0; i < 5 && scanner.Scan(); i++ {
			data := scanner.Text()
			receviedCsv := ConvertStringToArray(data)
			if i == 0 {
				continue
			}
			chunkOfData[i] = receviedCsv
		}

		// write data
		for _, item := range chunkOfData {
			number, _ := strconv.ParseFloat(item[0], 64)
			vr.smgr.Tick()

			metrics := Metrics{
				Values: []float64{
					fixedPrecision(float64(number)/1024/1024, 2),
					fixedPrecision(float64(number)/1024/1024, 2),
				},
				Time: memstats.T,
			}

			bs, _ := json.Marshal(metrics)
			w.Write(bs)
		}

	}

}

func ConvertStringToArray(rawCsv string) []string {
	array := strings.Split(rawCsv, ",")
	return array
}
