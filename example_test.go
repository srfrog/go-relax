// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax_test

import (
	"."
	"log"
	"net/http"
	"strconv"
)

type User struct {
	ID   int
	Name string
}

type Users struct {
	Group  string `item:"user"`
	People []*User
}

func (u *Users) List(rw relax.ResponseWriter, re *relax.Request) {
	println("users.List")
	rw.Header().Set("X-Something", "something goes here")
	rw.Respond(u)
}

func (u *Users) Create(rw relax.ResponseWriter, re *relax.Request) {
	println("users.Create")
	name := re.PostFormValue("name")
	if name != "" {
		user := &User{}
		user.ID = len(u.People) + 1
		user.Name = name
		u.People = append(u.People, user)
		return
	}
	rw.Error(http.StatusBadRequest, "not enough parameters to create; must supply a name")
}

func (u *Users) Read(rw relax.ResponseWriter, re *relax.Request) {
	println("users.Read")
	id, _ := strconv.Atoi(re.PathValues.Get("id"))
	for _, user := range u.People {
		if id == user.ID {
			rw.Respond(user)
			return
		}
	}
	rw.Error(http.StatusNotFound, "that user was not found")
}

func (u *Users) Update(rw relax.ResponseWriter, re *relax.Request) {
	rw.Error(http.StatusNotImplemented, "no Update yet!")
}

func (u *Users) Delete(rw relax.ResponseWriter, re *relax.Request) {
	rw.Error(http.StatusNotImplemented, "no Delete yet!")
}

// SampleHandler simply prints out all path values, and responds with a
// cheery message.
func SampleHandler(rw relax.ResponseWriter, re *relax.Request) {
	relax.Log.Println(relax.LOG_INFO, "SampleHandler", "Request:", re.Method, re.URL.Path)
	rw.Respond(re.PathValues)
}

// This example creates a new service under /api and serves requests
// to an users resource.
func Example_basic() {
	// set our log level to DEBUG for more detail
	relax.Log.SetLevel(relax.LOG_DEBUG)

	users := &Users{Group: "Influential Scientists"}

	// fill-in the list with some scientists
	users.People = append(users.People, &User{1, "Issac Newton"})
	users.People = append(users.People, &User{2, "Albert Einstein"})
	users.People = append(users.People, &User{3, "Nikola Tesla"})
	users.People = append(users.People, &User{4, "Charles Darwin"})
	users.People = append(users.People, &User{5, "Neils Bohr"})

	// create a service under /api/
	svc := relax.NewService("/api")

	// service-level filters (these could go inside NewService())
	svc.Filter(&relax.CORSFilter{
		AllowAnyOrigin:   true,
		AllowCredentials: true,
	})
	svc.Filter(&relax.OverrideFilter{})

	// I prefer pretty indentation.
	svc.Encoding(&relax.EncoderJSON{Indented: true})

	// serve our resource with CRUD routes, using unsigned ints as ID's.
	// this resource has SecurityFilter as resource-level filter.
	res := svc.Resource(users, &relax.SecurityFilter{}).CRUD("uint:id")

	// some other misc. routes to test route expressions.
	// these routes will be added under "/api/users/"
	res.Route("GET", "stuff/{date:date}", SampleHandler)
	res.Route("GET", "stuff/{int:int}", SampleHandler)
	res.Route("GET", "stuff/{hex:hex}", SampleHandler)
	res.Route("GET", "stuff/{word:word}", SampleHandler)
	res.Route("GET", "stuff/{whatever}/*", SampleHandler)
	// this one has a route-level filter
	res.POST("auth", SampleHandler, &relax.AuthBasic{})
	// this one is more specific
	res.GET("stats/{date:d1}/to/{date:d2}", SampleHandler)

	// let http.ServeMux handle basic routing.
	http.Handle(svc.Handler())

	log.Fatal(http.ListenAndServe(":3001", nil))
	// Output:
}
