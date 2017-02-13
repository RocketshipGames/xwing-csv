package main

import (
	"time"
	"strconv"
	"net/http"
	"encoding/json"
	"github.com/BellerophonMobile/logberry"
	"os"
	"strings"
	"fmt"
	"io/ioutil"
)

const ShipStatsURL = "https://github.com/guidokessels/xwing-data/raw/master/data/ships.js"
const PilotStatsURL = "https://github.com/guidokessels/xwing-data/raw/master/data/pilots.js"
const TournamentsFolder = "tournaments/"

//
// Various exceptions are required to make the different data sources
// together.  Search for "EXCEPTIONS" to find them all.
//

var actions = []string {
	"Focus",
	"Target Lock",
	"Barrel Roll",
	"Evade",
	"Boost",
	"Cloak",
	"SLAM",
	"Rotate Arc",
	//// Huge ships
	// "Coordinate",
	// "Jam",
	// "Recover",
	// "Reinforce",
}

var slots = []string{
	"Elite",
	"Astromech",
	"Salvaged Astromech",
	"Crew",
	"System",
	"Tech",
	"Turret",
	"Torpedo",
	"Missile",
	"Cannon",
	"Bomb",
	"Illicit",
	//// Huge ships
	// "Cargo",
	// "Hardpoint",
	// "Team",
	//// Ubiquitous
	// "Title",
	// "Modification",
}

func main() {
	defer logberry.Std.Stop()

	err := getshipstats()
	if err != nil {
		logberry.Main.Error(err)
		return
	}
	
	err = getpilotstats()
	if err != nil {
		logberry.Main.Error(err)
		return
	}

	err = gettournamentstats()
	if err != nil {
		logberry.Main.Error(err)
		return
	}

	writeshipstats()

	writeduplicatepilots()
	writepilotstats()
	
	writeliststats()

	logberry.Main.Info("Counts", logberry.D{
		"AllTime": alltimecounts,
		"Recent": recentcounts,
	})

}

func csvtext(s string) string {
	return "\"" + strings.Replace(s, "\"", "\"\"", -1) + "\""
}

type DataCounts struct {
	Tournaments int
	ListInstances int
	PilotInstances int
}

var alltimecounts DataCounts
var recentcounts DataCounts

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

type Flags map[string]int

func (f Flags) Check(flag string) string {

	_,ok := f[flag]

	if ok {
		if strings.Contains(flag, " ") {
			return csvtext(flag)
		}
		
		return flag
	}

	return ""
	
}

func (f Flags) Count(flag string) int {
	return f[flag]	
}

func (f Flags) Add(flag string) {
	c := f[flag]
	f[flag] = c+1	
}

func NewFlags(list []string) Flags {
	m := make(Flags)
	for _,s := range(list) {
		m.Add(s)
	}
	return m

}

func keyfields(keys []string) string {

	var s string = keys[0]

	for _,k := range(keys[1:]) {
		s = s + "," + k
	}

	return s
	
}

func keycheck(flags Flags, keys []string) string {

	var s string = flags.Check(keys[0])

	for _,k := range(keys[1:]) {
		s = s + "," + flags.Check(k)
	}

	return s
	
}

func keycount(flags Flags, keys []string) string {

	var s string = fmt.Sprint(flags.Count(keys[0]))

	for _,k := range(keys[1:]) {
		s = s + "," + fmt.Sprint(flags.Count(k))
	}

	return s
	
}

func ifbool(v bool, field string) string {
	if v {
		return field
	}
	return ""
}

type intwrapper int
func (i *intwrapper) UnmarshalJSON(data []byte) error {

	if v, err := strconv.Atoi(string(data)); err == nil {
		(*i) = intwrapper(v)
    return nil
  }

	(*i) = -1

  return nil
	
}

func factionmap(faction string) (string,error) {

	faction = strings.ToLower(faction)
	
	switch faction {
	case "rebel": fallthrough
	case "rebel alliance": fallthrough
	case "resistance":
		return "rebel",nil
		
	case "imperial": fallthrough
	case "galactic empire": fallthrough
	case "first order":
		return "imperial",nil

	case "scum": fallthrough
	case "scum and villainy":
		return "scum",nil		
	}

	return "", fmt.Errorf("Unknown faction %v", faction)
		
}

func shipmap(faction string, ship string) (string,error) {

	// BEGIN EXCEPTIONS
	ship = strings.ToLower(ship)

	ship = strings.Replace(ship, "adv.", "advanced", -1)
	
	switch ship {
	case "yt2400freighter":
		ship = "yt2400"
	}
	// END EXCEPTIONS
	
	f,err := factionmap(faction)
	return f + "/" + ship,err

}

func pilotmap(faction string, ship string, pilot string) (string,error) {

	// BEGIN EXCEPTIONS
	pilot = strings.ToLower(pilot)

	switch pilot {
	case "ltlorrir":
		pilot = "lieutenantlorrir"

	case "blackeightsqpilot":
		pilot = "blackeightsquadronpilot"

	case "sabinewren-swx56":
		pilot = "sabinewren"

	case "outerrimsmuggler":
		ship = "yt1300outerrimsmuggler"
	}
	// END EXCEPTIONS
	
	s,err := shipmap(faction,ship)
	return s + "/" + pilot,err

}

type Ship struct {
	Name string
	Faction []string
	Attack int
	Agility int
	Hull int
	Shields int
	Actions []string
	Maneuvers [][]int
	Size string
	XWS string
}

var shiplist []*Ship
var shipsXWS = make(map[string]*Ship)
var shipsGK = make(map[string]*Ship)

func getshipstats() error {

	task := logberry.Main.Task("Get ship stats")

	err := getasjson(&shiplist, ShipStatsURL, task)
	if err != nil {
		return task.Error(err)
	}

	// BEGIN EXCEPTIONS
	outerrimsmuggler := Ship{
		Name: "YT-1300 (Outer Rim Smuggler)",
		Faction: []string{ "Rebel Alliance" },
		Attack: 2,
		Agility: 1,
		Hull: 6,
		Shields: 4,
		Actions: []string{"Focus", "Target Lock"},
		Maneuvers: [][]int {
			[]int{0, 0, 0, 0, 0, 0},
			[]int{1, 2, 2, 2, 1, 0},
			[]int{1, 1, 2, 1, 1, 0},
			[]int{0, 1, 1, 1, 0, 3},
			[]int{0, 0, 1, 0, 0, 3},
		},
		Size: "large",
		XWS: "yt1300outerrimsmuggler",
	}
	shiplist = append(shiplist, &outerrimsmuggler)
	// END EXCEPTIONS
	
	// Process each ship
	for _,ship := range(shiplist) {

		factions := make(Flags)
		for _,x := range(ship.Faction) {
			fx,err := factionmap(x)
			if err != nil {
				return task.Error(err, ship)
			}
			factions.Add(fx)
		}
		
		for faction := range(factions) {
			code,err := shipmap(faction, ship.XWS)
			if err != nil {
				return task.Error(err)
			}
			if s,ok := shipsXWS[code]; ok {
				return task.Failure("Duplicate ship XWS", logberry.D{"Code": code, "New": ship, "Existing": s})
			}
			shipsXWS[code] = ship

			code,err = shipmap(faction, ship.Name)
			if err != nil {
				return task.Error(err)
			}
			if s,ok := shipsGK[code]; ok {
				return task.Failure("Duplicate ship name", logberry.D{"Code": code, "New": ship, "Existing": s})
			}
			shipsGK[code] = ship

		}

	}
	
	return task.Success(logberry.D{"Ships": len(shiplist)})

}

func writeshipstats() error {

	task := logberry.Main.Task("Write ship stats")

	f, err := os.Create("ships.csv")
	if err != nil {
		return task.Error(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "Name,Rebel,Imperial,Scum,Size,Attack,Agility,Hull,Shields,%v,XWS\n", keyfields(actions))
	for _,ship := range(shiplist) {

		
		factions := make(Flags)
		for _,x := range(ship.Faction) {
			fx,err := factionmap(x)
			if err != nil {
				return task.Error(err, ship)
			}
			factions.Add(fx)
		}

		sactions := NewFlags(ship.Actions)
		
		fmt.Fprintf(f, "%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v\n",
			csvtext(ship.Name),
			factions.Check("rebel"),
			factions.Check("imperial"),
			factions.Check("scum"),
			ship.Size,
			ship.Attack,
			ship.Agility,
			ship.Hull,
			ship.Shields,
			keycheck(sactions,actions),
			ship.XWS)
	}
	
	return task.Success()

}

type Uses struct {
	Total int
	Worlds int
	Nationals int
	Regionals int
	Stores int
	Vassals int
	Other int
}

func (x *Uses) Increment(scope string) error {
			
	x.Total++
	
	switch strings.ToLower(scope) {
	case "world championship":
		x.Worlds++
	case "nationals":
		x.Nationals++
	case "regional":
		x.Regionals++
	case "store championship":
		x.Stores++
	case "vassal play":
		x.Vassals++
	case "other":
		x.Other++				
	default:
		return fmt.Errorf("Unknown tournament scope %v", scope)
	}

	return nil

}

type Pilot struct {
	Name string
	Unique bool
	Ship string
	Skill intwrapper
	Points intwrapper
	Slots []string
	Text string
	Image string
	Faction string
	XWS string
	ship *Ship
	uniqueXWS string
	alltime Uses
	recent Uses
}

var pilotlist []*Pilot
var pilotsXWS = make(map[string]*Pilot)
var pilotnames = make(map[string][]*Pilot)

func getpilotstats() error {

	task := logberry.Main.Task("Get pilot stats")
	
	err := getasjson(&pilotlist, PilotStatsURL, task)
	if err != nil {
		return task.Error(err)
	}

	for _,pilot := range(pilotlist) {

		shipgkcode,err := shipmap(pilot.Faction, pilot.Ship)
		if err != nil {
			return task.Error(err)
		}			

		// BEGIN EXCEPTIONS
		if pilot.XWS == "outerrimsmuggler" {
			task.Warning("SMUGGLER")
			pilot.Ship = "YT-1300 (Outer Rim Smuggler)"
			shipgkcode,err = shipmap(pilot.Faction, "YT-1300 (Outer Rim Smuggler)")
			if err != nil {
				return task.Error(err)
			}	
		}
		// END EXCEPTIONS
		
		ship,ok := shipsGK[shipgkcode]
		if !ok {
			return task.Failure("Pilot has no ship", logberry.D{"Pilot": pilot, "ShipGKCode": shipgkcode})
		}

		pilot.ship = ship

		xws,err := pilotmap(pilot.Faction, ship.XWS, pilot.XWS)
		if err != nil {
			return task.Error(err)
		}			

		if p,ok := pilotsXWS[xws]; ok {
			return task.Failure("Duplicate pilot XWS", logberry.D{"XWS": xws, "New": pilot, "Existing": p})
		}
		pilotsXWS[xws] = pilot
		pilot.uniqueXWS = xws
		pilotnames[pilot.Name] = append(pilotnames[pilot.Name], pilot)
	}

	return task.Success(logberry.D{"Ships": len(pilotlist)})
	
}

func writeduplicatepilots() error {

	task := logberry.Main.Task("Write duplicate pilots")

	d, err := os.Create("pilot-duplicates.csv")
	if err != nil {
		return task.Error(err)
	}
	defer d.Close()

	for k,l := range(pilotnames) {
		if len(l) <= 1 {
			continue
		}
		
		fmt.Fprintf(d, "%v", csvtext(k))

		for _,p := range(l) {
			fmt.Fprintf(d, ",%v,%v,%v", p.XWS, csvtext(p.ship.Name), p.ship.XWS)
		}

		fmt.Fprintln(d)
		
	}

	return task.Success()

}

func writepilotstats() error {
	
	task := logberry.Main.Task("Write pilot stats")
	
	f, err := os.Create("pilots.csv")
	if err != nil {
		return task.Error(err)
	}
	defer f.Close()

	fields := []string{
		"Name",
		"XWS",
		"Faction",
		"Ship",
		"Unique",
		"Size",
		"Points",
		"Skill",
		"Attack",
		"Agility",
		"Hull",
		"Shields",
		keyfields(slots),
		csvtext("Total All Time Uses"),
		csvtext("World Championship All Time Uses"),
		csvtext("Nationals All Time Uses"),
		csvtext("Regional All Time Uses"),
		csvtext("Store Championship All Time Uses"),
		csvtext("Vassal All Time Uses"),
		csvtext("Other All Time Uses"),
		csvtext("Total Recent Uses"),
		csvtext("World Championship Recent Uses"),
		csvtext("Nationals Recent Uses"),
		csvtext("Regional Recent Uses"),
		csvtext("Store Championship Recent Uses"),
		csvtext("Vassal Recent Uses"),
		csvtext("Other Recent Uses"),
	}
	fmt.Fprintln(f, strings.Join(fields, ","))

	for _,pilot := range(pilotlist) {

		// BEGIN EXCEPTIONS
		if pilot.ship.Size == "huge" {
			continue
		}

		if pilot.XWS == "nashtahpuppilot" {
			continue
		}
		// END EXCEPTIONS		
		
		faction,err := factionmap(pilot.Faction)
		if err != nil {
			return task.Error(err)
		}

		pslots := NewFlags(pilot.Slots)

		data := []interface{}{
			csvtext(pilot.Name),
			pilot.XWS,
			faction,
			pilot.Ship,
			ifbool(pilot.Unique, "unique"),
			pilot.ship.Size,
			pilot.Points,
			pilot.Skill,
			pilot.ship.Attack,
			pilot.ship.Agility,
			pilot.ship.Hull,
			pilot.ship.Shields,			
			keycount(pslots,slots),
			pilot.alltime.Total,
			pilot.alltime.Worlds,
			pilot.alltime.Nationals,
			pilot.alltime.Regionals,
			pilot.alltime.Stores,
			pilot.alltime.Vassals,
			pilot.alltime.Other,			
			pilot.recent.Total,
			pilot.recent.Worlds,
			pilot.recent.Nationals,
			pilot.recent.Regionals,
			pilot.recent.Stores,
			pilot.recent.Vassals,
			pilot.recent.Other,			
		}
		var line string = fmt.Sprint(data[0])
		for _,d := range(data[1:]) {
			line = line + "," + fmt.Sprint(d)
		}		
		fmt.Fprintln(f, line)

	}
		
	return task.Success()

}

type ListInstance struct {

	EventCountry string
	EventState string
	EventScope string
	EventDate string
	EventPlayers int
	EventRank int

	List *List

}

var lists = make([]*ListInstance, 0)

func gettournamentstats() error {

	task := logberry.Main.Task("Get tournament stats")
	
	files, err := ioutil.ReadDir(TournamentsFolder)
	if err != nil {
		return task.WrapError("Could not read tournaments folder", err)
	}

	for _, file := range(files) {
		err := readtournament(TournamentsFolder+file.Name(), task)
		if err != nil {
			return task.Error(err)
		}
	}
	
	return task.Success()

}

type Venue struct {
	Name string `json:"venue"`
	Country string
	City string
	State string
}

type Upgrades struct {
	Elite []string `json:"ept"`
	Astromech []string `json:"amd"`
	SalvagedAstromech []string `json:"samd"`
	Crew []string
	System []string
	Tech []string
	Turret []string
	Torpedo []string
	Missile []string
	Cannon []string
	Bomb []string
	Illicit []string
	Cargo []string
	Hardpoint []string
	Team []string
	Title []string
	Modification []string `json:"mod"`
}

type PilotInstance struct {
	XWS string `json:"name"`
	Ship string
	Upgrades Upgrades
	pilot *Pilot	
}

type List struct {
	Faction string
	Pilots []*PilotInstance
}

type Rank struct {
	Swiss int
	Elimination int
}

type Player struct {
	List *List
	Rank Rank
}

type Tournament struct {
	Name string
	Date string
	Scope string `json:"type"`
	Format string
	PlayerCount int `json:"participant_count"`
	Venue Venue
	RoundDuration int `json:"round_length"`
	Players []Player
}


func errorline(js string, err error) int {

	var offset int64

	switch t := err.(type) {
	case *json.SyntaxError:
		offset = t.Offset
	case *json.UnmarshalTypeError:
		offset = t.Offset
	default:
		return -1
	}
	
	line := strings.Count(js[:offset], "\n")

	return line+1
	
}

func readtournament(file string, parent *logberry.Task) error {

	task := parent.Task("Read tournament", logberry.D{"File": file})

	// Read the previously downloaded tournament report
	bits, err := ioutil.ReadFile(file)
	if err != nil {
		return task.Error(err)
	}

	// Unmarshal the tournament report
	var fetch = struct{
		Tournament *Tournament // Temporary structure to match file format
	}{}
	
	err = json.Unmarshal(bits, &fetch)
	if err != nil {
		return task.Error(err, logberry.D{"Error": fmt.Sprintf("%T", err), "Line": errorline(string(bits), err)})
	}
	tournament := fetch.Tournament

	// Bail if there are no players reported
	if len(tournament.Players) <= 0 {
		task.Warning("Tournament has no players")
		return task.Success()
	}
	
	// Some tournaments don't report lists for all players, others don't
	// fill in the player count explicitly
	if tournament.PlayerCount < len(tournament.Players) {
		if tournament.PlayerCount != 0 {
			task.Warning("Under-reported player count", logberry.D{"Count": tournament.PlayerCount, "Reported": len(tournament.Players)})
		}
		tournament.PlayerCount = len(tournament.Players)
	}

	// Only tabulate standard tournaments
	if strings.ToLower(tournament.Format) != "standard - 100 point dogfight" {
		task.Warning("Not a dogfight tournament", tournament.Format)
		return task.Success()
	}
	
	// Determine whether or not this is a recent tournaments (past 4 months)
	recent := false
	date, err := time.Parse("2006-01-02", tournament.Date)
	if err == nil {
		if date.After(time.Now().AddDate(0,-4,0)) {
			recent = true
			task.Warning("Recent event!")
		}
	} else {
		task.Warning("Could not parse tournament date", err)
	}
	
	listcount := 0 // Count how many lists were reported for this tournament
	
	// For each player in the tournament	
	for _,player := range(tournament.Players) {

		// No list reported
		if player.List == nil {
			continue
		}

		if len(player.List.Pilots) <= 0 {
			task.Warning("Player has list but no pilots")
			continue
		}

		// For each pilot this player fielded
		for _,pilotinstance := range(player.List.Pilots) {
			xws,err := pilotmap(player.List.Faction, pilotinstance.Ship, pilotinstance.XWS)
			if err != nil {
				return task.Error(err)
			}

			pilot,ok := pilotsXWS[xws]
			if !ok {
				return task.Failure("Unknown pilot", logberry.D{"XWS": xws})
			}

			pilotinstance.pilot = pilot

			// Increment total times this pilot has been used
			err = pilot.alltime.Increment(tournament.Scope)
			if err != nil {
				return task.Error(err)
			}
			alltimecounts.PilotInstances++

			// Increment times this pilot has been used recently
			if recent {
				err = pilot.recent.Increment(tournament.Scope)
				if err != nil {
					return task.Error(err)
				}

				recentcounts.PilotInstances++				
			}

		}
		
		// Increment number of player lists reported
		if recent {
			recentcounts.ListInstances++
		}
		alltimecounts.ListInstances++

		// Create a list record
		listinstance := ListInstance{
			EventCountry: tournament.Venue.Country,
			EventState: tournament.Venue.State,
			EventScope: tournament.Scope,
			EventDate: tournament.Date,
			EventPlayers: tournament.PlayerCount,
			EventRank: player.Rank.Swiss,
			List: player.List,
		}
		lists = append(lists, &listinstance)
		
		listcount++
		
	}

	// Only count tournaments that actually reported players with lists
	if listcount > 0 {
		if recent {
			recentcounts.Tournaments++
		}
		alltimecounts.Tournaments++
	} else {
		task.Warning("No lists reported")
	}
	
	return task.Success()
	
}


type ListStats struct {
	
	SumShipPoints int
	
	NumShips int
	NumUniques int
	NumLarge int
	NumSmall int

	SumSkill int
	SumAttack int
	SumAgility int
	SumHull int
	SumShields int

	Text string

}

func NewListStats(list *List, parent *logberry.Task) (*ListStats,error) {

	task := parent.Task("Calculate list stats")

	var x ListStats
	
	for _,pilotinstance := range(list.Pilots) {
		pilot := pilotinstance.pilot
		
		x.SumShipPoints += int(pilot.Points)
		x.SumSkill += int(pilot.Skill)
		
		x.SumAttack += pilot.ship.Attack
		x.SumAgility += pilot.ship.Agility
		x.SumHull += pilot.ship.Hull
		x.SumShields += pilot.ship.Shields
			
		if pilot.Unique {
			x.NumUniques++
		}
			
		switch pilot.ship.Size {
		case "large":
			x.NumLarge++
		case "small":
			x.NumSmall++
		default:
			return nil,task.Failure("Unknown ship size", pilot.ship.Size)
		}


		descrip := pilotname(pilotinstance)
		
		if x.Text != "" {
			x.Text = x.Text + ", "
		}
		x.Text = x.Text + descrip

	}

	return &x,task.Success()

}

func pilotname(pilotinstance *PilotInstance) string {

	pilot := pilotinstance.pilot
	
	label := pilot.Name

	// BEGIN EXCEPTIONS
	//
	// This is a bit of a mess because the duplicates have slightly
	// different problems.  Some have different XWS but same ship name,
	// while others duplicate XWS but different ships.
	if len(pilotnames[pilot.Name]) > 1 {
		switch pilot.Name {
		case "Chewbacca": fallthrough
		case "Poe Dameron": fallthrough
		case "Han Solo":
			label = label + " (" + pilot.Faction + ")"			

		default:
			label = label + " (" + pilot.Ship + ")"
		}
	}
	// END EXCEPTIONS

	return label
	
}

func writeliststats() error {

	task := logberry.Main.Task("Write list stats")

	f, err := os.Create("lists.csv")
	if err != nil {
		return task.Error(err)
	}
	defer f.Close()

	fields := []string{
		"Date",
		"Scope",
		"Country",
		"State",
		csvtext("# Players"),
		"Rank",
		"Faction",
		csvtext("Ship Points"),
		csvtext("# Ships"),
		csvtext("# Uniques"),
		csvtext("# Large"),
		csvtext("# Small"),
		"Skill",
		"Attack",
		"Agility",
		"Hull",
		"Shields",
		"List",
	}
	fmt.Fprintln(f, strings.Join(fields, ","))

	for _,list := range(lists) {

		stats,err := NewListStats(list.List, task)
		if err != nil {
			return task.Error(err)
		}
		
		data := []interface{}{
			fmt.Sprintf("%v", csvtext(list.EventDate)),
			fmt.Sprintf("%v", csvtext(list.EventScope)),
			fmt.Sprintf("%v", csvtext(list.EventCountry)),
			fmt.Sprintf("%v", csvtext(list.EventState)),
			list.EventPlayers,
			list.EventRank,
			list.List.Faction,
			stats.SumShipPoints,
			len(list.List.Pilots),
			stats.NumUniques,
			stats.NumLarge,
			stats.NumSmall,
			stats.SumSkill,
			stats.SumAttack,
			stats.SumAgility,
			stats.SumHull,
			stats.SumShields,
			csvtext(stats.Text),
		}
		var line string = fmt.Sprint(data[0])
		for _,d := range(data[1:]) {
			line = line + "," + fmt.Sprint(d)
		}		
		fmt.Fprintln(f, line)

	}
	
	return task.Success()

}
