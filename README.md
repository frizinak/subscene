![screenshot](https://raw.githubusercontent.com/frizinak/subscene/dev/shot.png)

```
Usage of subscene
subscene [opts] <media query> <subtitle query>
  -i	run interactively instead of picking the first result
  -l string
    	subtitle language (default "english")
  -q	sush

<title query>:
    The media title to query subscene.com for.

<subtitle query>:
    The subtitle query we will try to find the best fuzzy match for.
    If this path exists and
         is a directory: the directory name will be used as query
                         and the subtitles will be unzipped here.
         is a file:      the filename without extension will be used as query
                         and only the first subtitle will be stored with the same
                         filename + '.srt'.
                         e.g.: subscene 'line of duty second' ~/owneddvdrips/line-of-duty-s02e03.avi
                               should result in ~/owneddvdrips/line-of-duty-s02e03.srt

```
