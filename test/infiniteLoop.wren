import "os" for Process
System.print("Now proceding to run infinite loop!")
Fiber.new {
    while(true) {
        System.print("loop")
        Process.sleep(100)
    }
}.try()
System.print("this line should be unreachable")