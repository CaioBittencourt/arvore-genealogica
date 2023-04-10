# Arvore genealógica
Desafio stone árvore genealógica
## Contents
* [What is it?](#what-is-it)
* [Endpoints](#endpoints)
* [Docs](#docs)
* [Run tests](#run-tests)
* [Start server in development mode](#start-server-in-development-mode)
* [Start server in production mode](#start-server-in-production-mode)

## What is it?
This is an application that builds a family tree to extract relationships amongst other things!


This application uses MongoDB replica set to leverage atomic transactions and uses `$graphLookup` to recursively search hierarchical data.
This application has two relationships on a database level. Each person can connect to each other through: `Children`, `Parent`. That being said i added the `Spouse` relationship to the domain layer because this relationship should affect the bacon number. To affect the bacons number you have to change how the family graph is connected. It wasnt very clear wether i should return spouses of the searched person, so the API currently returns the spouses for the search person!


The relationships that are identifiable in this application are:
- parent
- child
- sibling
- nephew
- cousin
- spouse
- aunt/uncle

That being said, what is returned by the API is ALL ascendants of a user plus siblings, nephew, uncles/aunts, spouses (of the searched person) and children.

## Endpoints

* POST - /person  => Stores a person
* GET - /person/:id/tree => Gets the family tree of a person
* GET - /person/:id/baconNumber/:id2 => Gets the bacon number between two persons
* GET - /person/:id/relationship/:id2 => Gets the relationship between two persons

## Docs

To document the API was used a library called: `https://github.com/swaggo/gin-swagger`, that generated the OpenAPI specification. To access the docs just use `make server-prod` and access: http://localhost:8080/swagger/index.html#

## Run tests
`make test`

## Start server in development mode
`make server-dev`

## Start server in production mode
`make server-prod`