# Arvore genealógica
Desafio stone árvore genealógica
## Contents
* [What is it?](#what-is-it)
* [Endpoints](#endpoints)
* [Run tests](#run-tests)
* [Start server in development mode](#start-server-in-development-mode)
* [Start server in production mode](#start-server-in-production-mode)

## What is it?
This is an application that builds a family tree to extract relationships amongst other things!

This application uses MongoDB replica set to leverage atomic transactions and uses `$graphLookup` to recursively search hierarquical data.
This application has three relationships on a database level. Each person can connect to eachother through: `Children`, `Parent` and `Spouse`. The reason i added the `Spouse` relationship its because this relationship should affect the bacon number. To affect the bacons number you have to change how the family graph is connected. Today, you cant pass Spouse as a relationship to the API, this relationship is inferred by inserted `Person`!

The relationships that are identifiable in this application are:
- parent
- child
- sibling
- nephew
- cousin
- spouse
- aunt/uncle

That being said, what is returned by the API is ALL ascendants of a user plus siblings, nephew, uncles/aunts, spouses and children.

## Endpoints

* POST - /person  => Stores a person
* GET - /person/:id/tree => Gets the family tree of a person
* GET - /person/:id/baconNumber/:id2 => Gets the bacon number between two persons
* GET - /person/:id/relationship/:id2 => Gets the relationship between two persons

## Run tests
`make test`

## Start server in development mode
`make server-dev`

## Start server in production mode
`make server-prod`

