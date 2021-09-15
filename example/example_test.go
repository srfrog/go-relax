// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax_test

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/srfrog/go-relax"
	"github.com/srfrog/go-relax/filter/authbasic"
	"github.com/srfrog/go-relax/filter/cors"
	"github.com/srfrog/go-relax/filter/etag"
	"github.com/srfrog/go-relax/filter/gzip"
	"github.com/srfrog/go-relax/filter/logs"
	"github.com/srfrog/go-relax/filter/override"
	"github.com/srfrog/go-relax/filter/security"
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

// FindByID searches users.People for a user matching ID and returns it;
// or StatusError if not found. This could do a search in our DB and
// handle the error logic.
func (u *Users) FindByID(idstr string) (*User, error) {
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return nil, &relax.StatusError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	for _, user := range u.People {
		if id == user.ID {
			// user found, return it.
			return user, nil
		}
	}
	// user not found.
	return nil, &relax.StatusError{Code: http.StatusNotFound, Message: "That user was not found"}
}

// Index handles "GET /v1/users"
func (u *Users) Index(ctx *relax.Context) {
	ctx.Header().Set("X-Custom-Header", "important header info from my framework")
	// list all users in the resource.
	ctx.Respond(u)
}

// Create handles "POST /v1/users"
func (u *Users) Create(ctx *relax.Context) {
	user := &User{}
	// decode json payload from client
	if err := ctx.Decode(ctx.Request.Body, &user); err != nil {
		ctx.Error(http.StatusBadRequest, err.Error())
		return
	}
	// some validation
	if user.Name == "" {
		ctx.Error(http.StatusBadRequest, "must supply a name")
		return
	}
	if user.DOB.IsZero() {
		user.DOB = time.Now() // lies!
	}
	// create new user
	user.ID = len(u.People) + 1
	u.People = append(u.People, user)
	// send restful response
	ctx.Respond(user, http.StatusCreated)
}

// Read handles "GET /v1/users/ID"
func (u *Users) Read(ctx *relax.Context) {
	user, err := u.FindByID(ctx.PathValues.Get("id"))
	if err != nil {
		ctx.Error(err.(*relax.StatusError).Code, err.Error(), "more details here")
		return
	}
	ctx.Respond(user)
}

// Update handles "PUT /v1/users/ID" for changes to items.
func (u *Users) Update(ctx *relax.Context) {
	user, err := u.FindByID(ctx.PathValues.Get("id"))
	if err != nil {
		ctx.Error(err.(*relax.StatusError).Code, err.Error(), "more details here")
		return
	}
	// maybe some validation should go here...

	// decode json payload from client
	if err := ctx.Decode(ctx.Request.Body, &user); err != nil {
		ctx.Error(http.StatusBadRequest, err.Error())
		return
	}
	ctx.Respond(user)
}

// Delete handles "DELETE /v1/users/ID" to remove items.
// Note: this function wont be used because we override the route below.
func (u *Users) Delete(ctx *relax.Context) {
	ctx.Error(http.StatusInternalServerError, "not reached!")
}

// SampleHandler prints out all filter info, and responds with all path values.
func SampleHandler(ctx *relax.Context) {
	ctx.Respond(ctx.PathValues)
}

// Example_basic creates a new service under path "/v1" and serves requests
// for the users resource.
func Example_basic() {
	// Create our resource object.
	users := &Users{Group: "Influential Scientists"}

	// Fill-in the users.People list with some scientists (this could be from DB table).
	users.People = []*User{
		&User{1, "Issac Newton", time.Date(1643, 1, 4, 0, 0, 0, 0, time.UTC)},
		&User{2, "Albert Einstein", time.Date(1879, 3, 14, 0, 0, 0, 0, time.UTC)},
		&User{3, "Nikola Tesla", time.Date(1856, 7, 10, 0, 0, 0, 0, time.UTC)},
		&User{4, "Charles Darwin", time.Date(1809, 2, 12, 0, 0, 0, 0, time.UTC)},
		&User{5, "Neils Bohr", time.Date(1885, 10, 7, 0, 0, 0, 0, time.UTC)},
	}

	// Create a service under "/v1". If using absolute URI, it will limit requests
	// to a specific host. This service has FilterLog as service-level filter.
	svc := relax.NewService("/v1", &logs.Filter{})

	// More service-level filters (these could go inside NewService()).
	svc.Use(&etag.Filter{}) // ETag with cache conditionals
	svc.Use(&cors.Filter{
		AllowAnyOrigin:   true,
		AllowCredentials: true,
	})
	svc.Use(&gzip.Filter{})     // on-the-fly gzip encoding
	svc.Use(&override.Filter{}) // method override support

	// I prefer pretty indentation.
	json := relax.NewEncoder()
	json.Indented = true
	svc.Use(json)

	// Basic authentication, used as needed.
	needsAuth := &authbasic.Filter{
		Realm: "Masters of Science",
		Authenticate: func(user, pass string) bool {
			if user == "Pi" && pass == "3.14159" {
				return true
			}
			return false
		},
	}

	// Serve our resource with CRUD routes, using unsigned ints as ID's.
	// This resource has Security filter at resource-level.
	res := svc.Resource(users, &security.Filter{CacheDisable: true}).CRUD("{uint:id}")
	{
		// Although CRUD added a route for "DELETE /v1/users/{uint:id}",
		// we can change it here and respond with status 418.
		teapotted := func(ctx *relax.Context) {
			ctx.Error(418, "YOU are the teapot!", []string{"more details here...", "use your own struct"})
		}
		res.DELETE("{uint:id}", teapotted)

		// Some other misc. routes to test route expressions.
		// Shese routes will be added under "/v1/users/".
		res.GET("dob/{date:date}", SampleHandler)               // Get by ISO 8601 datetime string
		res.PUT("issues/{int:int}", SampleHandler)              // PUT by signed int
		res.GET("apikey/{hex:hex}", res.NotImplemented)         // Get by APIKey (hex value) - 501-"Not Implemented"
		res.GET("@{word:word}", SampleHandler)                  // Get by username (twitterish)
		res.GET("stuff/{whatever}/*", teapotted)                // sure, stuff whatever...
		res.POST("{uint:id}/checkin", SampleHandler, needsAuth) // POST with route-level filter
		res.GET("born/{date:d1}/to/{date:d2}", SampleHandler)   // Get by DOB in date range
		res.PATCH("", res.MethodNotAllowed)                     // PATCH method is not allowed for this resource.

		// Custom regexp PSE matching.
		// Example matching US phone numbers. Any of these values are ok:
		// +1-999-999-1234, +1 999-999-1234, +1 (999) 999-1234, 1-999-999-1234
		// 1 (999) 999-1234, 999-999-1234, (999) 999-1234
		res.GET(`phone/{re:(?:\+?(1)[\- ])?(\([0-9]{3}\)|[0-9]{3})[\- ]([0-9]{3})\-([0-9]{4})}`, SampleHandler)
		// Example matching month digits 01-12
		res.GET(`todos/month/{re:([0][1-9]|[1][0-2])}`, SampleHandler)

		// New internal method extension (notice the X).
		res.Route("XMODIFY", "properties", SampleHandler)
	}

	// Let http.ServeMux handle basic routing.
	http.Handle(svc.Handler())

	log.Fatal(http.ListenAndServe(":8000", nil))
	// Output:
}
