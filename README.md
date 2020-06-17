# yapperbot-frs
Bot that powers the [Feedback Request Service](https://en.wikipedia.org/wiki/WP:FRS) on Wikipedia

## Pruning the FRS list
To remove users who haven't made any contributions in over three years, you can run the "prune" command:
`yapperbot-frs --prune --dbserver [ip:port] --dbuser [username] --db [dbname] > prune.txt`
You must have a connection to a replica of the enwiki database to do this, the details of which you should put into the command. You will be asked to enter the password into stdin.

prune.txt will be formed of two parts: the updated wikitext, and a list of users who have been pruned. The intent here is that the list can be used for a mass message to notify them that they are falling off the list.
