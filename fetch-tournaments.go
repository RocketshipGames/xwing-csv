package main

import (
	"encoding/json"
	"os"
	"github.com/BellerophonMobile/logberry"
	"fmt"
	"net/http"
	"io/ioutil"
//	"time"
)

const ListJugglerAPI = "http://lists.starwarsclubhouse.com/api/v1/"
const TournamentListURL = ListJugglerAPI + "tournaments"
const TournamentsFolder = "tournaments/"

func main() {
	defer logberry.Std.Stop()

	// Create directory for the results
	err := os.MkdirAll(TournamentsFolder, 0755)
	if err != nil {
		logberry.Main.WrapError("Could not create tournaments folder", err, logberry.D{"Folder": TournamentsFolder})
		return
	}

	// Fetch the list of tournaments
	var tournaments = struct {
		Tournaments []int
	}{}

	task := logberry.Main.Task("Get tournament list")
	err = getasjson(&tournaments, TournamentListURL, task)
	if err != nil {
		task.Error(err)
		return
	}
	task.Success(logberry.D{"Count": len(tournaments.Tournaments)})

	errlog, err := os.Create("tournaments.errors")
	if err != nil {
		task.Error(err)
		return
	}
	defer errlog.Close()

	// Fetch all the listed tournaments
	for _,id := range(tournaments.Tournaments) {
		task := logberry.Main.Task("Get tournament", logberry.D{"ID": id})
		
		url := fmt.Sprintf("%vtournament/%v", ListJugglerAPI, id)
		err := download(fmt.Sprintf("%v%v.json", TournamentsFolder, id), url, task)
		if err != nil {
			fmt.Fprintln(errlog, url)
			task.Error(err)
			continue
		}

		task.Success()

	}
	
}

func download(dest string, url string, parent *logberry.Task) error {

	task := parent.Task("Download", logberry.D{"URL": url, "Destination": dest})

	resp,err := http.Get(url)
	if err != nil {
		return task.Error(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return task.WrapError("Could not read body", err)
	}
	
	if resp.StatusCode != 200 {
		return task.Failure("Server error", logberry.D{"Status": resp.StatusCode, "Response": string(body)})
	}

	err = ioutil.WriteFile(dest, body, 0777)
	if err != nil {
		return task.WrapError("Could write body", err)
	}
	
	return task.Success()
	
}

func getasjson(dest interface{}, url string, parent *logberry.Task) error {

	task := parent.Task("Get as JSON", logberry.D{"URL": url, "Type": fmt.Sprintf("%T", dest)})

	resp,err := http.Get(url)
	if err != nil {
		return task.Error(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return task.WrapError("Could not read body", err)
	}
	
	if resp.StatusCode != 200 {
		return task.Failure("Server error", logberry.D{"Status": resp.StatusCode, "Response": string(body)})
	}

	err = json.Unmarshal(body, dest)
	if err != nil {
		return task.WrapError("Could unmarshal body", err)
	}
	
	return task.Success()

}
