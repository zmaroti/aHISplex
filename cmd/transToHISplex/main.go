// tool to convert phased genotypes from a bcftools query to the
// HIrisPlex-S uploadable .csv format.
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

type marker struct {
    id          string
    forward     bool
    testAllele  string
}

// marker data in the order of expected .csv input
var markerData []marker

// hash map of chromosome:pos to markerdata indexes
// used to map vcf positions to hplex markers
var idxMap  map[string]int

// reading the translate table, returning the
// header line of the .csv file as a string
func readTranslate(fn string) string {
    inFile, err := os.Open(fn)
    defer inFile.Close()

    if err != nil {
        panic(err)
    }

    idxMap = make(map[string]int)

    scanner := bufio.NewScanner(inFile)
    scanner.Split(bufio.ScanLines)

    var idx int

    var markerIds []string

    for scanner.Scan() {
        // rs1126809_A	11	89017961	F	A
        arr := strings.Split(scanner.Text(), "\t")

        if len(arr) != 5 {
            fmt.Fprintf(os.Stderr, "Invalid line: %s\n", scanner.Text())
            os.Exit(1)
        }

        chrpos := fmt.Sprintf("%s:%s", arr[1], arr[2])


        var m marker
        m.id = arr[0]

        if arr[3] == "F" {
            m.forward = true
        } else if arr[3] != "R" {
            fmt.Fprintf(os.Stderr, "Invalid line: %s\n", scanner.Text())
            os.Exit(1)
        }

        m.testAllele = arr[4]

        markerData = append(markerData, m)
        markerIds  = append(markerIds, arr[0])
        idxMap[chrpos] = idx

        idx++
    }

    if scanner.Err() != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    return fmt.Sprintf("sampleid,%s", strings.Join(markerIds, ","))
}

// parsing a bcftools query format to its component
// data is expected as series of
//     sampleID=phased_Genotype
// separated by tabs
func getData(line string, idx int) []string {
    arr := strings.Split(line, "\t")
    data := make([]string, len(arr))

    for i, str := range arr {
        // SAMPLEID=GT
        sdarr := strings.Split(str, "=")

        data[i] = sdarr[idx]
    }

    return data
}

// helper function to convert int slice to string slice
func int2str (numbers []int) []string {
    ret := make([]string, len(numbers))

    for idx, n := range numbers {
        ret[idx] = fmt.Sprintf("%d", n)
    }

    return ret
}

// The nucleotid on the complement strand
func Complement(n string) string {
    switch strings.ToUpper(n) {
        case "A":
            return "T"
        case "T":
            return "A"
        case "G":
            return "C"
        case "C":
            return "G"
        default:
            fmt.Fprintf(os.Stderr, "Invalid nucleotid letter: %s\n", n)
            os.Exit(1)
    }

    return "" // stupid go
}

// NOTE, the bcf query is performed on GLIMPE2 imputed/phased bcf file
// due to how GLIMPSE2 works its if GRANTED that
// * ALL sites are genotyped
// * all sites are included that exist in the reference panel
// * all reference sites are BIALLELIC (no multiple allele condition)
// * only alleles existing in the biallelic REF panel are imputed
// ===> in our case it means, all minor/major of all HPLEX-S markers are genotyped
func parseBCFquery (fn, header string) {
    var scanner *bufio.Scanner

    if fn == "-" {
        scanner = bufio.NewScanner(os.Stdin)
    } else {
        inFile, err := os.Open(fn)
        defer inFile.Close()

        if err != nil {
            panic(err)
        }

        scanner = bufio.NewScanner(inFile)
    }

    buf := make([]byte, 1024*1024)

    scanner.Buffer(buf, bufio.MaxScanTokenSize)

    scanner.Split(bufio.ScanLines)

    var sampleList  []string
    var sampleData  [][]int
    var dataAlloc   bool
    var m marker

    for scanner.Scan() {
        arr := strings.SplitN(scanner.Text(), "\t", 5)

        // we have sample count/ids on the fly from the bcfquery output
        if !dataAlloc {
            // parse sampleIds from the bcf query line
            sampleList = getData(arr[4], 0)

            // 2d slice for sample/marker counts
            sampleData = make([][]int, len(sampleList))

            // slice for HISplex marker count data
            for i, _ := range sampleData {
                sampleData[i] = make([]int, len(idxMap))
            }

            dataAlloc  = true
        }

        // reference based position chr:pos
        chrpos := fmt.Sprintf("%s:%s", arr[0], arr[1])

        // HIrisplex-S marker index of the given marker
        if markerIdx, ok := idxMap[chrpos]; ok {
            var ref, alt    string
            var reftest bool

            m = markerData[markerIdx]

            if !m.forward {
                ref, alt = Complement(arr[2]), Complement(arr[3])
            } else {
                ref, alt = strings.ToUpper(arr[2]), strings.ToUpper(arr[3])
            }

            if m.testAllele == ref {
                reftest = true
            } else if m.testAllele == alt {
                reftest = false
            } else {
                var strand string

                if m.forward {
                    strand = "forward"
                } else {
                    strand = "reverse"
                }

                fmt.Fprintf(os.Stderr, "Invalid genotype for marker %s: %s (strand %s, ref: %s, alt: %s)",
                    m.id, m.testAllele, strand, arr[2], arr[3])

                os.Exit(1)
            }

            gts := getData(arr[4], 1)

            for sampleIdx, gt := range gts {
                var gtCount int

                // for het, gtcount is always 1 (we have only biallelic)
                if gt == "0|1" || gt == "1|0" {
                    gtCount = 1
                } else if gt == "0|0" {
                // our GLIMPSE2 phased data is true 1KG REF/ALT
                // the translate data contains whether we test on REF or ALT
                // VCF 0|0 is hom ref; 1|1 is hom alt
                // for hom alleles (ref or alt), we have to check if hplex test allele is ref or alt
                    if reftest {
                        gtCount = 2
                    } else {
                        gtCount = 0
                    }
                } else if gt == "1|1" {
                    if reftest {
                        gtCount = 0
                    } else {
                        gtCount = 2
                    }
                } else { // no other options, we have only biallelic data
                    fmt.Fprintf(os.Stderr, "Invalid genotype %s for marker %s: %s (strand %s, ref: %s, alt: %s)",
                        gt, m.id, m.testAllele, arr[2], arr[3])

                    os.Exit(1)
                }

                sampleData[sampleIdx][markerIdx] = gtCount
            }
        } else {
            fmt.Fprintf(os.Stderr, "Missing marker in HIrisplex-S data: %s\n", chrpos)
        }
    }

    // die on any error when reading the file
    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "Reading Stdin: ", err)

        os.Exit(1)
    }

    // print outpout to STDOUT, we use this tool from shell with piping
    fmt.Println(header)

    for sampleIdx, sample := range sampleList {
        fmt.Printf("%s,%s\n", sample, strings.Join(int2str(sampleData[sampleIdx]), ","))
    }
}

func printHelp() {
    fmt.Fprintln(os.Stderr,
`USAGE
transToHISplex <translatetable.tsv> <bcfquery.tsv|->

Translates the reference based bcftools query to the HIrisPlex-S '.csv'
format that can be upload to the HIrisPlex webtool and prints the
result to STDOUT.

The translate table is a tab separated file containing the rs ids
(in the order of the expected input .csv file), the chromosome/position
of the markers that is used to identify the variants from the phased
GLIMPSE2 vcf, the strand the variant is based on (F/R) and the test
allele of the HIrisPlex-S marker the allele counts are based on.

The second argument for the bcftools query is either a standard file
or also can be piped from STDIN (with the "-" option).
`)
    os.Exit(0)
}


func main() {
    if len(os.Args) != 3 {
        printHelp()
    }

    header := readTranslate(os.Args[1])

    parseBCFquery(os.Args[2], header)
}
