# teltech-coding-project

Write a simple web application in Go which accepts math problems via the URL and returns the response in JSON. The application should be simple to test using curl or wget.

Example:

http://localhost/add?x=2&y=5
Output:

{"action": "add", "x": 2, "y": 5, "answer", 7, "cached": false}
Implement add, subtract, multiply, and divide
Only two arguments will ever be passed -- x and y (no need to handle a variable number of arguments)
Cache results so that repeated calls with the same problem will return the answer from cache
Show in the output JSON whether the cache was used
