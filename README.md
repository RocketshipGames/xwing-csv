# Ship Data

This repository provides simple scripts to pull, fuse, and transform
data from [X-Wing Data](https://github.com/guidokessels/xwing-data)
and [List Juggler](http://lists.starwarsclubhouse.com/), which are
linked by the [X-Wing Squadron
Specification](https://github.com/elistevens/xws-spec).  In
particular, it provides a tool to export pilot stats and usage data to
CSV so non-programmers can easily use their spreadsheet tool of choice
to incorporate some quantitative analysis into their list building.

## Scripts

The tools are all written as Go programs intended to be used in script
fashion, i.e.:

    % go run script.go

### fetch-tournaments.go

This retrieves the current list of tournaments available in
ListJuggler, and then downloads all of them into the `tournaments/`
folder.  Note that a good portion of them will fail with an internal
server error.  The cause of this is currently unknown.

### csv-compile.go

This compiles ship and pilot stats from X-Wing Data and usage data
from ListJuggler into a simple CSV format.  The most recent X-Wing
Data is pulled directly from its repository.  The script assumes that
the `fetch-tournaments.go` script has been previously used to pull
down tournament data from ListJuggler.

The script creates the following CSV files:

  * `ships.csv`: All the nominal ships stats and properties.
    Technically stats are associated with specific pilot cards, but in
    reality there is only one pilot with different stats for its
    class: The [Outer Rim
    Smuggler](http://xwing-miniatures.wikia.com/wiki/Outer_Rim_Smuggler),
    a lesser version of the YT-1300.  So most people tend to think in
    terms of ship chassis and associated stats, which are presented
    here.  The Smuggler is included as a separate entry.

  * `pilots.csv`: All of the pilots in the game, their ship stats, and
    counts breaking down all the times that pilot has been used in a
    list captured in ListJuggler.
    
    For simplicity, the compilation excludes:

    ** Epic Play: Tournaments for Epic games are ignored, as are the
       huge ships since they're only for Epic play and complicate
       analysis with fore & aft sections.

    ** The Nashtah Pup: It's not fieldable on its own.

  * `lists.csv`: Summaries of all the lists captured in ListJuggler.
    The core of this are summed stats needed to do some [simple
    analysis](http://www.rocketshipgames.com/blogs/tjkopena/2016/12/x-wing-beginner-squad-building/)
    based on raw attacks, agility, hull points+shields, and the number
    of ships.

The script also generates `pilot-duplicates.csv`, but this is only for
development purposes (there are several duplicate entities following
the XWS, which this output presents to enable deconfliction).

## License

These tools and resources are provided under the open source
[MIT license](http://opensource.org/licenses/MIT):

> The MIT License (MIT)
>
> Copyright (c) 2017 [Joe Kopena](http://rocketshipgames.com/blogs/tjkopena/)
> 
>
> Permission is hereby granted, free of charge, to any person
> obtaining a copy of this software and associated documentation files
> (the "Software"), to deal in the Software without restriction,
> including without limitation the rights to use, copy, modify, merge,
> publish, distribute, sublicense, and/or sell copies of the Software,
> and to permit persons to whom the Software is furnished to do so,
> subject to the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
> BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
> ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
> CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
> SOFTWARE.
