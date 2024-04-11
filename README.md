# infer-jailer-restrictions

This algorithm is part of an effort to automate the generation of extraction models for [Jailer](https://github.com/Wisser/Jailer).
I developed it because a model with restrictions as provided by [Jailer](https://github.com/Wisser/Jailer) (where all the has-child associations/green arrows are disabled) is insufficient for my use case. The reason being that I want to start my subset from a table that is referenced by many other tables (i.e. has many children), and actually references few tables (i.e. has few parents).

As an example, my subject table represents conferences and I want to extract all the information related to a particular conference - attendees, tickets, talks, etc. Each of these entities are modelled by a table that contains a foreign key referencing the conference to which they “belong”.

---
Now onto the algorithm. The goal is to infer **which associations should be enabled and disabled**. It starts by building a directed graph where the nodes are tables and the edges are associations. This is just to have efficient access to the parents and the children of a given table.

Then, starting from the subject tables, it will maintain two sets of tables: `visited` tables and the current `frontier`.
`visited` represents the tables that were already processed. The `frontier` represents tables that will be visited next, and a table `T` is added to the frontier when we infer that an association between a visited table and `T` must be enabled (more on that below).

Processing a table is deciding which associations involving that table must be enabled. There are two slightly different phases where we process a table:

### Process a subject table
Here, all the _has-parent_ associations will be enabled and their inverse will be disabled. Consequently, all the parent tables will be added to the `frontier`. Besides that, we will also enable all the _has-child_ associations - this is how we include the entities that “belong to” the subject entity in the subset. Note that on this step we also disable the inverse of these _has-child_ associations; this is discussed below. As with the parents, all the children of the subject table are added to the `frontier`.

### Process a table in the frontier
Again, all the _has-parent_ associations will be enabled and their inverse will be disabled. The difference now is how we process the _has-child_ associations.

The children of `frontier` tables don’t directly belong to the initial subject tables, so we are not obligated to enable these _has-child_ associations. The only case in which we should enable these associations is if there is no direct way (i.e. transitively using has-parent associations) to reach those children tables. From my experience, this will either be a join table representing a _m:n_ relation or a table that represents the private attributes of its parent.

At the moment of processing the current table, we don’t know whether its children are “directly accessible” or not. As such, we just record the association in a set of `unexploredEdges`. This set is only processed when there are no more tables to visit in the frontier. Processing this set may add more tables to the frontier, so the algorithm is enclosed in an loop that only ends when it’s not possible to add more tables to the frontier - either from has-parent associations or from processing `unexploredEdges`.

--- 
One potentially unsafe detail about this algorithm is that we disable the inverse of the enabled _has-children_ associations (i.e. we are disabling actual _has-parent_ dependencies). In my case I had to do it because I had some foreign key inconsistencies that were causing the subset to explode. I am not sure if this is problematic or not, and in a database with no fk inconsistencies it is safer not to do it.
