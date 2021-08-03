foreign class File {
    construct open(path, flags, permissions) {}
    foreign read(count)
}

foreign class Stat {

}

foreign class Process {
    foreign static sleep(delay)
}