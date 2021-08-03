import "extras" for extras
import "os" for File, Stat

System.print("Hello world")
System.print(extras)

System.print("Loading \"test/main.wren\"")
var file = File.open("test/main.wren", 0, 511)


System.print("Printing \"test/main.wren\"")
System.print(file.read(0))

System.print("Loading \"Not Exist\"")
var error = Fiber.new {
    File.open("Not Exist", 0, 511)
}.try()

System.print("Successfully caught error: " + error)

var fn = Fn.new {|data|
    System.print("Hello world!")
    System.print("Data passed in is: " + data)
}