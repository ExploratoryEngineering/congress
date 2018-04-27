package main

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"fmt"
	"math/rand"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/server"
	"github.com/ExploratoryEngineering/congress/storage"
	"github.com/ExploratoryEngineering/logging"
)

var lastnames = []string{
	"Moore", "Franklin", "Frey", "Lester", "Lang", "Sutton", "Compton",
	"Arroyo", "Huerta", "Howard", "White", "Valencia", "Ramos", "Pope",
	"Burch", "Downs", "Moreno", "Salazar", "Charles", "Massey", "Olsen",
	"Schneider", "Ferguson", "Clements", "Liu", "Ortiz", "Garza", "Frazier",
	"Washington", "Tucker", "Mclaughlin", "Chase", "Petersen", "Faulkner",
	"Jenkins", "Turner", "Shelton", "Frederick", "Burton", "Figueroa",
	"Gordon", "Marshall", "Robertson", "Khan", "Vazquez", "Benson",
	"Nguyen", "Hudson", "Jensen", "Stevens", "Dunlap", "Neal", "Curry",
	"Caldwell", "Morgan", "Prince", "Hanson", "Leblanc", "Randolph",
	"Archer", "Noble", "Garrison", "Conner", "Ali", "Aguirre", "Shaw",
	"French", "Martin", "Mendez", "Brewer", "Dawson", "Stephens", "Stafford",
	"Knight", "Whitney", "Thomas", "Poole", "Dennis", "Nelson", "Wise",
	"Mcgrath", "Lawrence", "Forbes", "Hernandez", "Patel", "Ellison",
	"Novak", "Harris", "Leon", "Beltran", "Stephenson", "Patton",
	"Mcdonald", "Combs", "Molina", "Waller", "Johnson", "Mccarthy",
	"Ashley", "Wade",
}
var firstnames = []string{
	"Rylan", "Kale", "Curtis", "Giovanny", "Noe", "Chaim", "Adriel",
	"Trenton", "Sincere", "Augustus", "Zaire", "Colton", "Wilson",
	"Tristian", "Wade", "Dax", "Conner", "Warren", "Kyler", "Jackson",
	"Kyan", "Emerson", "Tate", "Manuel", "Mason", "Jaeden", "Saul",
	"Hunter", "Brenton", "Aarav", "Larry", "Corbin", "Roberto", "Albert",
	"Yahir", "Brayan", "Damarion", "Quincy", "Dante", "Jason", "Dillon",
	"Peter", "Bobby", "Elliott", "Damien", "Teagan", "Allen", "Malachi",
	"Braedon", "Dayton", "Pamela", "Jaylene", "Mckinley", "Tia", "Kamari",
	"Gracelyn", "Kiley", "Amy", "Esperanza", "Penelope", "Lizeth",
	"Madison", "Aspen", "Margaret", "Lina", "Mina", "Zoie", "Ariana",
	"Laila", "Carina", "Daphne", "Casey", "Sara", "Kyra", "Anahi",
	"Emilia", "Crystal", "Kiara", "Giada", "Sloane", "Adalynn", "Carly",
	"Yazmin", "Kayley", "Hailie", "Zaria", "Katie", "Alia", "Myah",
	"Miracle", "Emery", "Tatiana", "Itzel", "Eliana", "Lindsay", "Nola",
	"June", "Rory", "Emerson", "Erika",
}

func makeRandomName() string {
	return firstnames[rand.Int31n(int32(len(firstnames)))] + " " + lastnames[rand.Int31n(int32(len(lastnames)))]
}

func makeRandomEmail() string {
	return fmt.Sprintf("%08x@example.com", rand.Int63())
}

func createRandomUser() model.User {
	ret := model.User{
		ID:    model.UserID(fmt.Sprintf("%08x", rand.Int63())),
		Name:  makeRandomName(),
		Email: makeRandomEmail(),
	}
	return ret
}

func generateUsers(count int, datastore storage.Storage, keyGen *server.KeyGenerator, callback func(user model.User)) {

	for i := 0; i < count; i++ {
		user := createRandomUser()
		if err := datastore.UserManagement.AddOrUpdateUser(user, (*keyGen).NewID); err != nil {
			logging.Error("Unable to generate user: %v", err)
		} else {
			callback(user)
		}
		logging.Info("Created user %s (%s)", user.Name, user.ID)
	}
	logging.Info("Generated %d users", count)
}
