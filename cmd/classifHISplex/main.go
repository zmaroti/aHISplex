// tool to annotate HIrisPlexS result csv file with the eye, hair and skin predictions
// based on the p values and the guide (Walsh et al 2014)
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "strconv"
    "sort"
    "flag"
)

type sortable struct {
	pVals   []float64
    idxs    []int
}

func (s sortable) Len() int           { return len(s.pVals) }
func (s sortable) Less(i, j int) bool { return s.pVals[i] > s.pVals[j] }
func (s sortable) Swap(i, j int) {
	s.pVals[i], s.pVals[j] = s.pVals[j], s.pVals[i]
	s.idxs[i], s.idxs[j] = s.idxs[j], s.idxs[i]
}

func strs2floats(arr []string) []float64 {
    ret := make([]float64, len(arr))

    var err error
    var f float64

    for i, str := range arr {
        f, err = strconv.ParseFloat(str, 64)

        if err != nil {
            panic(err)
        }

        ret[i] = f
    }

    return ret
}

func classifEye(arr []string) string {
    // PBlueEye, PIntermediateEye, PBrownEye
    // highest is eye color
    pVals := strs2floats(arr)

    if pVals[0] > pVals[1] && pVals[0] > pVals[2] {
        return "Blue"
    }

    if pVals[1] > pVals[0] && pVals[1] > pVals[2] {
        return "Intermediate"
    }

    return "Brown"
}

func classifHair(arr1, arr2 []string) string {
    // order is BLOND, BROWN, RED, BLACK
    pVals      := strs2floats(arr1)

    // order is LIGHT, DARK
    pLightDark := strs2floats(arr2)
    
    // pBlackHair is highest
    if pVals[3] > pVals[0] && pVals[3] >= pVals[1] && pVals[3] > pVals[2] {
        if pVals[3] > 0.7 {
            return "Black"
        } else if pLightDark[1] > 0.5 {
            return "Black"
        }
        return "Dark brown/Black"
    }

    // pBlondHair is highest
    if pVals[0] >= pVals[1] && pVals[0] > pVals[2] && pVals[0] > pVals[3] {
        if pVals[0] > 0.7 {
            if pLightDark[0] > 0.95 {
                return "Blond"
            }
            return "Blond/Dark-Blond"
        } else if pLightDark[0] > 0.9 {
            return "Blond/Dark-Blond"
        }
        return "Dark-Blond/Brown"
    }

    // pBrownHair is highest
    if pVals[1] >= pVals[0] && pVals[1] > pVals[2] && pVals[1] > pVals[3] {
        if pVals[1] > 0.7 {
            if pLightDark[0] > 0.8 {
                return "Brown"
            }
            return "Brown/Dark-Brown"
        } else if pLightDark[0] > 0.8 {
            return "Brown/Dark-Brown"
        }
        return "Dark-Brown/Black"
    }

    // pRedHair is highest
    if pVals[2] > pVals[0] && pVals[2] > pVals[1] && pVals[2] > pVals[3] {
        return "Red"
    }

    // shouldn't get here
    return "Invalid hair color"
}

func classifSkin(arr []string) string {
    // order of the array indexes
    // 0             1         2                 3         4
    // PVeryPaleSkin,PPaleSkin,PIntermediateSkin,PDarkSkin,PDarktoBlackSkin
    pVals := strs2floats(arr)

    idxs := make([]int, len(pVals))
    for i := range idxs {
        idxs[i] = i
    }

    d := sortable{pVals, idxs}

    sort.Sort(d)

    if d.pVals[0] > 0.9 {
        switch d.idxs[0] {
            case 0:
                return "Very Pale"
            case 1:
                return "Pale"
            case 2:
                if d.pVals[1] == 4 {
                    return "Dark"
                } else {
                    return "Intermediate"
                }
            case 3:
                return "Dark"
            case 4:
                return "Dark-Black"
        }
    } else if d.pVals[0] > 0.7 { // first p > 0.7
        switch d.idxs[0] {
            case 0: // first Very Pale
                if d.pVals[1] < 0.15 { // second < 0.15
                    return "Very Pale"
                }
                return "Very Pale/Darker" // second > 0.15
            case 1: // first Pale
                if d.pVals[1] < 0.15 {  // second < 0.15
                    return "Pale"
                }
                if d.idxs[1] == 0 {  // second = Very Pale, p > 0.15
                    return "Pale/Lighter"
                }
                return "Pale/Darker"  // second darker than Pale, p >0.15

            case 2: // first Intermediate
                if d.pVals[1] < 0.15 { // second p<0.15
                    return "Intermediate"
                }
                if d.idxs[1] < 2 {  // second lighter than Intermediate, p > 0.15
                    return "Intermediate/Lighter"
                }
                return "Intermediate/Darker" // second darker than Intermediate, p > 0.15

            case 3: // first Dark
                if d.idxs[1] == 4 { // second is Dark-Black
                    return "Dark-Black"
                }
                return "Dark"  // second is lighter than Dark

            case 4:
                return "Dark-Black" // first is Dark-Black
        }
    } else {
        switch d.idxs[0] {
            case 0:
                if d.idxs[1] == 1 {
                    return "Pale/Lighter"
                } else if d.idxs[1] == 2 {
                    return "Intermediate/Lighter"
                }
                return "Dark/Lighter"

            case 1:
                if d.idxs[1] == 0 {
                    return "Pale"                 // second is very pale
                } else if d.idxs[1] == 2 {
                    return "Intermediate/Lighter" // second is intermediate
                }
                return "Dark/Lighter"

            case 2:
                if d.idxs[1] > 2 {
                    return "Intermediate/Darker"
                }

                return "Intermediate"

            case 3:
                if d.idxs[1] == 4 {
                    return "Dark-Black/Dark"
                }
                return "Dark"

            case 4:
                if d.idxs[1] == 3 {
                    return "Dark-Black/Dark"
                }
                return "Dark-Black"
        }
    }

    return "Invalid skin color"
}

// reads the result file and classifies the eye, hair and skin colors based on the p values
func classifResults(fn string, short bool) {
    inFile, err := os.Open(fn)
    defer inFile.Close()

    if err != nil {
        panic(err)
    }

    scanner := bufio.NewScanner(inFile)
    scanner.Split(bufio.ScanLines)

    scanner.Scan()

    if scanner.Err() != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    if short {
        fmt.Println("sampleid,EyeColor,HairColor,SkinColor")
    } else {
        fmt.Printf("%s,%s\n", scanner.Text(), "EyeColor,HairColor,SkinColor")
    }

    for scanner.Scan() {
        arr := strings.Split(scanner.Text(), ",")

        if len(arr) != 90 {
            fmt.Fprintf(os.Stderr, "Invalid result line; 90 fields are expected, and only %d were found\n%s",
                len(arr), scanner.Text())

            os.Exit(1)
        }

        var eyeColor, hairColor, skinColor string
        
        eyeColor  = classifEye(arr[42:45]) 
        hairColor = classifHair(arr[53:57], arr[67:70])
        skinColor = classifSkin(arr[73:78])

        if short {
            fmt.Printf("%s,%s,%s,%s\n", arr[0], eyeColor, hairColor, skinColor)
        } else {
            fmt.Printf("%s,%s,%s,%s\n", strings.Join(arr, ","), eyeColor, hairColor, skinColor)
        }
    }

    if scanner.Err() != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func printHelp() {
    fmt.Fprintln(os.Stderr,
`USAGE
classifHISplex [-short] <HIrisPlex-S_result.csv>

Translates the Result.csv outpiut file from the HIrisPlex-S website containing
the p values for the predicted phenotypes to the eye, hair, skin colors and
prints the result on the screen (STDOUT).

If the -short option is used then only the sample IDs and the predicted
phenotypes are displayed. By default all the supporting data with the
additional 3 predicted phenotype columns are displayed.
`)
    os.Exit(0)
}


func main() {
    var help, short bool

    flag.BoolVar(&help,   "help", false, "print help")
    flag.BoolVar(&short,   "short", false, "Only print the sample id and the classification without the supporting data.")

    flag.Parse()

    args := flag.Args()

    if help || (len(args) != 1) {
        printHelp()
    }

    classifResults(args[0], short)
}
