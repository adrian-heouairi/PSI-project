package main
import "fmt"

func displayMenuAndTakeChoice(choices []string) (int, error) {
    for i, value :=range choices {
        fmt.Println(i + 1, " - ", value)
    }
    var choice int
    fmt.Print("[", 1, "..", len(choices), "] : ")
    _, err := fmt.Scanf("%d",&choice)
    if err != nil {
        fmt.Println(err)
        return -1, err
    }
    return choice,nil
}

func mainMenu() {
    choices := []string{"Show available peers", "Show addresses of peer", "Show tree of peer", "Dump full peer tree"}
    choice ,err := displayMenuAndTakeChoice(choices)
    fmt.Println(choice,err)
    switch choice {
    case 1:
        restGetPeers(true)
    case 2:
        peers, err := restGetPeers(false)
        if err != nil {
           return
        }
        choice, err = displayMenuAndTakeChoice(peers)
        restGetAddressesOfPeer(peers[choice - 1],true)
    case 3: 

        peers, err := restGetPeers(false)
        if err != nil {
           return
        }
        choice, err = displayMenuAndTakeChoice(peers)
        listAllFilesOfPeer(peers[choice - 1])
        // TODO CONTINUE THIS CASE
    case 4:
        peers, err := restGetPeers(false)
        if err != nil {
           return
        }
        choice, err = displayMenuAndTakeChoice(peers)
        downloadFullTreeOfPeer(peers[choice - 1])
    }
}
