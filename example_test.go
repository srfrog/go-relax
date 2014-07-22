// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax_test

import (
	"github.com/codehack/go-relax"
	"log"
	"net/http"
	"strconv"
	"time"
)

// User could be a struct mapping a DB table.
type User struct {
	ID   int       `json:"id"`
	Name string    `json:"name"`
	DOB  time.Time `json:"dob"`
}

// Users will be our resource object.
type Users struct {
	Group  string  `json:"group"`
	People []*User `json:"people"`
}

// FindById searches users.People for a user matching ID and returns it;
// or StatusError if not found. This could do a search in our DB and
// handle the error logic.
func (u *Users) FindById(idstr string) (*User, error) {
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return nil, &relax.StatusError{http.StatusInternalServerError, err.Error(), nil}
	}
	for _, user := range u.People {
		if id == user.ID {
			// user found, return it.
			return user, nil
		}
	}
	// user not found.
	return nil, &relax.StatusError{http.StatusNotFound, "That user was not found", nil}
}

// List handles "GET /api/users"
func (u *Users) List(rw relax.ResponseWriter, re *relax.Request) {
	rw.Header().Set("X-Custom-Header", "important header info from my framework")
	// list all users in the resource.
	rw.Respond(u)
}

// Create handles "POST /api/users"
func (u *Users) Create(rw relax.ResponseWriter, re *relax.Request) {
	user := &User{}
	// decode json payload from client
	if err := re.Decode(re.Body, &user); err != nil {
		rw.Error(http.StatusBadRequest, err.Error())
		return
	}
	// some validation
	if user.Name == "" {
		rw.Error(http.StatusBadRequest, "must supply a name")
		return
	}
	if user.DOB.IsZero() {
		user.DOB = time.Now() // lies!
	}
	// create new user
	user.ID = len(u.People) + 1
	u.People = append(u.People, user)
	// send restful response
	rw.Respond(user, http.StatusCreated)
}

// Read handles "GET /api/users/ID"
func (u *Users) Read(rw relax.ResponseWriter, re *relax.Request) {
	user, err := u.FindById(re.PathValues.Get("id"))
	if err != nil {
		rw.Error(err.(*relax.StatusError).Code, err.Error(), "more details here")
		return
	}
	rw.Respond(user)
}

// Update handles "PUT /api/users/ID" for changes to items.
func (u *Users) Update(rw relax.ResponseWriter, re *relax.Request) {
	user, err := u.FindById(re.PathValues.Get("id"))
	if err != nil {
		rw.Error(err.(*relax.StatusError).Code, err.Error(), "more details here")
		return
	}
	// maybe some validation should go here...

	// decode json payload from client
	if err := re.Decode(re.Body, &user); err != nil {
		rw.Error(http.StatusBadRequest, err.Error())
		return
	}
	rw.Respond(user)
}

// Delete handles "DELETE /api/users/ID" to remove items.
// Note: this function wont be used because we override the route below.
func (u *Users) Delete(rw relax.ResponseWriter, re *relax.Request) {
	rw.Error(http.StatusInternalServerError, "not reached!")
}

// SampleHandler prints out all filter info, and responds with all path values.
func SampleHandler(rw relax.ResponseWriter, re *relax.Request) {
	relax.Log.Println(relax.LOG_INFO, "SampleHandler", "Request:", re.Method, re.URL.Path)
	re.Info.Print() // print info passed down by filters
	rw.Respond(re.PathValues)
}

// Example_basic creates a new service under path "/api" and serves requests
// for the users resource.
func Example_basic() {
	// set our log level to DEBUG for more detail
	relax.Log.SetLevel(relax.LOG_DEBUG)

	// create our resource
	users := &Users{Group: "Influential Scientists"}

	// fill-in the users.People list with some scientists (this could be from DB table)
	users.People = []*User{
		&User{1, "Issac Newton", time.Date(1643, 1, 4, 0, 0, 0, 0, time.UTC)},
		&User{2, "Albert Einstein", time.Date(1879, 3, 14, 0, 0, 0, 0, time.UTC)},
		&User{3, "Nikola Tesla", time.Date(1856, 7, 10, 0, 0, 0, 0, time.UTC)},
		&User{4, "Charles Darwin", time.Date(1809, 2, 12, 0, 0, 0, 0, time.UTC)},
		&User{5, "Neils Bohr", time.Date(1885, 10, 7, 0, 0, 0, 0, time.UTC)},
	}

	// create a service under /api/
	svc := relax.NewService("/api", &relax.FilterGzip{}, &relax.FilterETag{})

	// service-level filters (these could go inside NewService())
	svc.Filter(&relax.FilterCORS{
		AllowAnyOrigin:   true,
		AllowCredentials: true,
	})
	// method override support
	svc.Filter(&relax.FilterOverride{})

	// I prefer pretty indentation.
	svc.Encoding(&relax.EncoderJSON{Indented: true})

	// Basic authentication, used as needed
	needsAuth := &relax.FilterAuthBasic{
		Realm: "Masters of Science",
		Authenticate: func(user, pass string) bool {
			if user == "Pi" && pass == "3.14159" {
				return true
			}
			return false
		},
	}

	// serve our resource with CRUD routes, using unsigned ints as ID's.
	// this resource has FilterSecurity as resource-level filter.
	res := svc.Resource(users, &relax.FilterSecurity{CacheDisable: true}).CRUD("{uint:id}")

	// although CRUD added a route for "DELETE /api/users/{uint:id}",
	// we can override it here and respond with status 418.
	teapotted := func(rw relax.ResponseWriter, re *relax.Request) {
		rw.Error(418, "YOU are the teapot!", []string{"more details here...", "use your own struct"})
		// or using rw.Respond():
		// rw.Respond("YOU are the teapot!", 418)
	}
	res.DELETE("{uint:id}", teapotted)

	// some other misc. routes to test route expressions.
	// these routes will be added under "/api/users/"
	res.GET("dob/{date:date}", SampleHandler)               // Get by ISO 8601 datetime string
	res.PUT("{int:int}", SampleHandler)                     // PUT by signed int
	res.GET("apikey/{hex:hex}", SampleHandler)              // Get by APIKey (hex value)
	res.GET("@{word:word}", SampleHandler)                  // Get by username (twitterish)
	res.GET("stuff/{whatever}/*", teapotted)                // sure, stuff whatever...
	res.POST("{uint:id}/checkin", SampleHandler, needsAuth) // POST with route-level filter
	res.GET("born/{date:d1}/to/{date:d2}", SampleHandler)   // Get by DOB in date range

	// let http.ServeMux handle basic routing.
	http.Handle(svc.Handler())

	log.Fatal(http.ListenAndServe(":8000", nil))
	// Output:
}
