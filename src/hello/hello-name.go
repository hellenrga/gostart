package main

import (
	"fmt"
)

func main() {

	// In go you can declare variables without specifying the type.
	var name = "Hellen"

	// You can also omit the word 'var'. by adding := to declare the variable
	age := 19
	version := 1.22

	fmt.Println("Hi, ms.", name, ",your age is", age)
	fmt.Println("This program runs in version", version)

	/* 	fmt.Println("The type of the variable 'name' is: ", reflect.TypeOf(name))
	fmt.Println("The type of the variable 'age' is: ", reflect.TypeOf(age))
	fmt.Println("The type of the variable 'version' is: ", reflect.TypeOf(version)) */
}
