package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
type BookStore struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	BookName   string
	BookAuthor string
	BookISBN   string
	BookPages  int
	BookYear   int
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			BookName:   "The Vortex",
			BookAuthor: "José Eustasio Rivera",
			BookISBN:   "958-30-0804-4",
			BookPages:  292,
			BookYear:   1924,
		},
		{
			BookName:   "Frankenstein",
			BookAuthor: "Mary Shelley",
			BookISBN:   "978-3-649-64609-9",
			BookPages:  280,
			BookYear:   1818,
		},
		{
			BookName:   "The Black Cat",
			BookAuthor: "Edgar Allan Poe",
			BookISBN:   "978-3-99168-238-7",
			BookPages:  280,
			BookYear:   1843,
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"id":     res.ID.Hex(),
			"name":   res.BookName,
			"author": res.BookAuthor,
			"isbn":   res.BookISBN,
			"pages":  res.BookPages,
			"year":   res.BookYear,
		})
	}

	return ret
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://alibabaazimi:tSqT8sll6hMta6Hp@cc-exercise.2nboky2.mongodb.net/?retryWrites=true&w=majority&appName=cc-exercise"))

	// mongodb://localhost:27017
	// mongodb+srv://alibabaazimi:tSqT8sll6hMta6Hp@cc-exercise.2nboky2.mongodb.net/?retryWrites=true&w=majority&appName=cc-exercise

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")

	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(middleware.Logger())

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		authors := findAllBooks(coll)
		return c.Render(200, "author-table", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		years := findAllBooks(coll)
		return c.Render(200, "year-table", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.Render(200, "create-book", nil)
	})

	e.GET("/edit/:id", func(c echo.Context) error {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
		}

		var book BookStore
		if err = coll.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&book); err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}

		b := map[string]interface{}{
			"ID":         book.ID.Hex(),
			"BookName":   book.BookName,
			"BookAuthor": book.BookAuthor,
			"BookISBN":   book.BookISBN,
			"BookPages":  book.BookPages,
			"BookYear":   book.BookYear,
		}

		return c.Render(200, "edit-book", b)
	})

	e.GET("/api/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.JSON(http.StatusOK, books)
	})

	e.GET("/api/books/:id", func(c echo.Context) error {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			// return 299
			return c.JSON(http.StatusNotModified, map[string]string{"error": "invalid id"})
		}

		var book BookStore
		if err = coll.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&book); err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}

		book_str := map[string]interface{}{
			"id":     book.ID.Hex(),
			"book":   book.BookName,
			"author": book.BookAuthor,
			"isbn":   book.BookISBN,
			"pages":  book.BookPages,
			"year":   book.BookYear,
		}

		return c.JSON(http.StatusOK, book_str)
	})

	e.POST("/api/books", func(c echo.Context) error {
		if c.FormValue("name") == "" || c.FormValue("author") == "" || c.FormValue("isbn") == "" {
			return c.JSON(http.StatusNotModified, map[string]string{"error": "missing form data"})
		}

		book := BookStore{
			BookName:   c.FormValue("name"),
			BookAuthor: c.FormValue("author"),
			BookISBN:   c.FormValue("isbn"),
			BookPages:  func() int { i, _ := strconv.Atoi(c.FormValue("pages")); return i }(),
			BookYear:   func() int { i, _ := strconv.Atoi(c.FormValue("year")); return i }(),
		}

		books := findAllBooks(coll)
		// check if book already exists
		for _, b := range books {
			if b["isbn"] == book.BookISBN {
				// return 200
				return c.JSON(http.StatusOK, "book already exists")

			}

			if b["name"] == book.BookName && b["author"] == book.BookAuthor {
				// return 200
				return c.JSON(http.StatusOK, "book already exists")
			}
		}

		// insert book into database
		result, err := coll.InsertOne(context.TODO(), book)
		if err != nil {
			// return 304
			return c.JSON(http.StatusNotModified, map[string]string{"error": "book not created"})
		}

		// return 201
		return c.JSON(http.StatusCreated, "book created with id: "+result.InsertedID.(primitive.ObjectID).Hex())
	})

	e.PUT("/api/books", func(c echo.Context) error {
		id, err := primitive.ObjectIDFromHex(c.FormValue("id"))
		// if err != nil {
		// 	return c.JSON(http.StatusNotModified, map[string]string{"error": "invalid id"})
		// }

		// if c.FormValue("name") == "" || c.FormValue("author") == "" || c.FormValue("isbn") == "" || c.FormValue("pages") == "" || c.FormValue("year") == "" {
		// 	return c.JSON(http.StatusNotModified, map[string]string{"error": "missing form data"})
		// }

		book := BookStore{
			BookName:   c.FormValue("name"),
			BookAuthor: c.FormValue("author"),
			BookISBN:   c.FormValue("isbn"),
			BookPages:  func() int { i, _ := strconv.Atoi(c.FormValue("pages")); return i }(),
			BookYear:   func() int { i, _ := strconv.Atoi(c.FormValue("year")); return i }(),
		}

		// books := findAllBooks(coll)

		// check if book already exists
		// for _, b := range books {
		// 	if b["name"] == book.BookName && b["author"] == book.BookAuthor && b["isbn"] == book.BookISBN {
		// 		// return 200
		// 		return c.JSON(http.StatusOK, "book already exists")

		// 	}
		// }

		// update book in database
		_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": id}, bson.M{"$set": book})
		if err != nil {
			return c.JSON(http.StatusNotModified, map[string]string{"error": "book not updated"})
		}

		return c.JSON(http.StatusOK, "book updated")
	})

	// e.DELETE("/api/books/:id", func(c echo.Context) error {
	// 	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	// 	if err != nil {
	// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	// 	}

	// 	if _, err = coll.DeleteOne(context.TODO(), bson.M{"_id": id}); err != nil {
	// 		return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
	// 	}

	// 	return c.JSON(http.StatusOK, map[string]string{"message": "book deleted"})
	// })

	e.Logger.Fatal(e.Start(":3030"))
}
