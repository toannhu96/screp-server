package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/icza/screp/rep"
	"github.com/icza/screp/repparser"
)

var BadRequestErr = fmt.Errorf("bad request error")

func main() {
	r := gin.Default()
	r.GET("/process", func(c *gin.Context) {
		inputFile := c.Query("input")
		outputFile := c.Query("output")
		overview, err := strconv.ParseBool(c.DefaultQuery("overview", "true"))
		if err != nil {
			c.Error(BadRequestErr)
			return
		}
		if inputFile == "" || outputFile == "" {
			c.Error(BadRequestErr)
			return
		}
		process(inputFile, outputFile, overview)
		c.JSON(http.StatusOK, gin.H{
			"status": true,
		})
	})
	r.Run(":9091")
}

func process(infile string, outfile string, overview bool) {
	cfg := repparser.Config{
		Commands:    true,
		MapData:     true,
		MapGraphics: true,
		Debug:       true,
	}
	r, err := repparser.ParseFileConfig(infile, cfg)
	if err != nil {
		fmt.Printf("Failed to parse replay: %v\n", err)
		return
	}

	var destination = os.Stdout
	foutput, err := os.Create(outfile)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer foutput.Close()
	destination = foutput

	if overview {
		printOverview(r)
	}

	enc := json.NewEncoder(destination)
	var valueToEncode any = r
	custom := map[string]any{}
	// If there are custom data, wrap (embed) the replay in a struct that holds the custom data too:
	if len(custom) > 0 {
		valueToEncode = struct {
			*rep.Replay
			Custom map[string]any
		}{r, custom}
	}
	if err := enc.Encode(valueToEncode); err != nil {
		fmt.Printf("Failed to encode output: %v\n", err)
		return
	}
}

func printOverview(rep *rep.Replay) {
	rep.Compute()

	engine := rep.Header.Engine.ShortName
	if rep.Header.Version != "" {
		engine = engine + " " + rep.Header.Version
	}
	mapName := rep.MapData.Name
	if mapName == "" {
		mapName = rep.Header.Map // But revert to Header.Map if the latter is not available.
	}
	winner := ""
	if rep.Computed.WinnerTeam != 0 {
		winner = fmt.Sprint("Team ", rep.Computed.WinnerTeam)
	}

	fmt.Println("Engine  :", engine)
	fmt.Println("Date    :", rep.Header.StartTime.Format("2006-01-02 15:04:05 -07:00"))
	fmt.Println("Length  :", rep.Header.Frames.String())
	fmt.Println("Title   :", rep.Header.Title)
	fmt.Println("Map     :", mapName)
	fmt.Println("Type    :", rep.Header.Type.Name)
	fmt.Println("Matchup :", rep.Header.Matchup())
	fmt.Println("Winner  :", winner)

	fmt.Println("Team  R  APM EAPM   @  Name ")
	for i, p := range rep.Header.Players {
		pd := rep.Computed.PlayerDescs[i]
		mins := pd.LastCmdFrame.Duration().Minutes()
		var apm, eapm int
		if pd.CmdCount > 0 {
			apm = int(float64(pd.CmdCount)/mins + 0.5)
		}
		if pd.EffectiveCmdCount > 0 {
			eapm = int(float64(pd.EffectiveCmdCount)/mins + 0.5)
		}
		fmt.Println("%3d   %s %4d %4d  %2d  %s\n", p.Team, p.Race.Name[:1], apm, eapm, pd.StartDirection, p.Name)
	}
}
