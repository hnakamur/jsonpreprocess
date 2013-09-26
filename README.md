jsonpreprocess
==============

json preprocessor library and tools written in Go.

According to [JSON](http://json.org/), you cannot have comments in JSON.
However sometimes you would like to have comments in documents.

This preprocessor can do:

- trim comments
- minify (trim comments and whitespaces).


This preprocessor accepts two style of comments.

- line comment:
    from ```//``` to the end of line.
    example: ```// this is a comment```
- block comment:
    from ```/*``` to ```*/```. block comments cannot be nested.
    example: ```/* this is a comment */```


example input

```
/* response example */
{
  "error": false, // true: if error occurred, false: otherwise
  "count": 23, // item total count
  "items": [ // items in this page
    {
      "id": 1,
      "name": "John Doe" // user name
    },
    {
      "id": 2,
      "name": "Paul Smith"
    }
  ]
}
```
