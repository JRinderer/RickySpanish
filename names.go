package main

import (
	"math/rand"
	"time"
)

// Military operations from WWII and the Korean War
var operationNames = []string{
	// WWII - European Theater
	"Overlord", "Barbarossa", "MarketGarden", "Torch", "Husky",
	"Dragoon", "Bagration", "Dynamo", "Neptune", "Cobra",
	"Varsity", "Plunder", "Compass", "Crusader", "Battleaxe",
	"Jubilee", "Weserübung", "Cartwheel", "Bodenplatte", "Grenade",
	"Veritable", "Clipper", "Totalise", "Tractable", "Charnwood",
	"Epsom", "Goodwood", "Bluecoat", "Lüttich", "Spring",
	"Pluto", "Eclipse", "Sunrise", "Unthinkable", "Sealion",
	"Cerberus", "Catapult", "Ironclad", "Pedestal", "Lightfoot",
	"Supercharge", "Bertram", "Pugilist", "Vulcan", "Corkscrew",
	// WWII - Pacific Theater
	"Galvanic", "Flintlock", "Forager", "Stalemate", "Iceberg",
	"Detachment", "Coronet", "Olympic", "Downfall", "Hailstone",
	"Cartwheel", "Elkton", "Dexterity", "Persecution", "Reckless",
	"Typhoon", "Catchpole", "Desecrate", "Transom", "Inmate",
	"King", "Musketeer", "Victor", "Oboe", "Montclair",
	// Korean War
	"Chromite", "Thunderbolt", "Killer", "Ripper", "Rugged",
	"Dauntless", "Piledriver", "Strangle", "Saturate", "Showdown",
	"Smack", "ClamUp", "Counter", "LittleSwitch", "BigSwitch",
	"Courageous", "Tomahawk", "Commando", "Wolfhound", "Roundup",
	"Ripcord", "Minden", "Claymore", "Ratkiller", "Spooky",
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateOperationName() string {
	name := operationNames[rng.Intn(len(operationNames))]
	return "Operation" + name
}
