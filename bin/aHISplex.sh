#!/bin/sh

# setup the data dir we have the reference etc
BINDIR=$( dirname -- "$0"; )   # get script dir
DATA="$BINDIR/aHISplex_data"   # get data dir
# if you move this script to some other place, setup DATA path to the original aHISplex_Data directory
# DATA=/path/to/aHISplex_data

# at TOOL DETECTION section of the code we try to figure the executables in YOUR PATH
# HOWEVER, if it is impossible to put all the binaries in your path you can hardwire the exacutables
# NOTE due to difefrent versions of BOOST library in dynamically linked GLIMPSE2 executables the
# precompiled reference data can be incompatible. If you use the static libraries included in GLIMPSE2 v1.0
# then the precompiled data should be OK
#
# G2PHASE=/path/to/GLIMPSE2_phase_static
# G2LIGATE=/path/to/GLIMPSE2_ligate_static
# BCFTOOLS=/path/to/bcftools

# Usage info
show_help() {
cat << EOF
Usage: ${0##*/} [-h?ns] [-t threads] [-o OUTPREF|DEFAULT:impute] [-r reference|DEFAULT:GRCh37] <BAMFILE.bam|BAMLIST.txt>

Runs HIrisPlex-S analysis on a single bam file or a list of bamfiles. The markers are
genotyped by GLIMPSE2 using imputation based on the appropriate 1KG PhaseIII reference data set.
The 41 markers of the HirisPlex-S are filtered by bcftools. And the variants are translated
to HIrisPlex-S allele count by a custom tool.

optional arguments:
     -s             Silent mode, suppress verbose information on the analysis.
     -n             Don't use parallel execution  even if parallel tool is available.
     -t threads     try to use more threads (default: 1)
     -o outputdir   the output directory of the imputed data (default: impute)
     -r reference   the reference the BAM files are mapped to. NOTE you can't mix GRCh37 and hg19
                    BAM files due to differences in chromosome naming convention (1 vs chr1).
                    This is not an issue in case of GRCH38 vs hg38 BAM files.
EOF
}

# A POSIX variable
OPTIND=1         # Reset in case getopts has been used previously in the shell.

# Initialize our own variables
outdir="impute"
threads=1
parallel=1
verbose=1
ref="GRCh37"

# parse options
while getopts "h?nst:o:r:" opt; do
    case "$opt" in
    h|\?)
        show_help
        exit 0
        ;;
    n)  parallel=0
        ;;
    s)  verbose=0
        ;;
    t)  threads=$OPTARG
        ;;
    o)  outdir=$OPTARG
        ;;
    r)  ref=$OPTARG
        ;;
    *)  show_help >&2
        exit 1
        ;;
    esac
done

shift $((OPTIND-1))

# show help if we have no/more than 1 arguments
if [ $# -ne 1 ]; then
    show_help >&2
    exit 1
fi

#### TOOL DETECTION
# checking if we have the tools in our PATH
tool_ok() {
     type "$1" >/dev/null 2>&1
}

# prefer the static tool as the binary reference included in the package
# can break on dynamic compilation based on different boot library
if tool_ok "GLIMPSE2_phase_static"; then
    G2PHASE=$( type -P GLIMPSE2_phase_static )
elif tool_ok "GLIMPSE2_phase"; then
    if [ "$verbose" -eq 1 ]; then
       echo "WARNING: Using dynamic linked GLIMPSE2_phase. Note precompiled binary reference data could be incompatible if GLIMPSE2 was compiled with different version of boost library."
    fi
    G2PHASE=$( type -P GLIMPSE2_phase )
fi

if tool_ok "GLIMPSE2_ligate_static"; then
    G2LIGATE=$( type -P GLIMPSE2_ligate_static )
elif tool_ok "GLIMPSE2_ligate"; then
    if [ "$verbose" -eq 1 ]; then
       echo "WARNING: Using dynamically linked GLIMPSE2_ligate. Note precompiled binary reference could be incompatible if GLIMPSE2 was compiled with different version of boost library."
    fi
    G2LIGATE=$( type -P GLIMPSE2_ligate )
fi

if tool_ok "bcftools"; then
    BCFTOOLS=$( type -P bcftools )
fi

if [ -z "$G2PHASE" ]; then
    echo "ERROR: GLIMPSE2_phase must be in your PATH."
    exit 1
elif [ ! -x "$G2PHASE" ]; then
    echo "ERROR: $G2PHASE must be executable."
    exit 1
fi

if [ -z "$G2LIGATE" ]; then
    echo "ERROR: GLIMPSE2_ligate must be in your PATH."
    exit 1
elif [ ! -x "$G2LIGATE" ]; then
    echo "ERROR: $G2LIGATE must be executable."
    exit 1
fi

if [ -z "$BCFTOOLS" ]; then
    echo "ERROR: bcftools must be in your PATH."
    exit 1
elif [ ! -x "$BCFTOOLS" ]; then
    echo "ERROR: $BCFTOOLS must be in your PATH."
    exit 1
fi

#### END of tool detection


# setup a gracful exit if we receive SIGINT/SIGKILL
do_for_sigint() {
    echo "! Received SIGINT, exiting"
    exit 1
}

trap 'do_for_sigint' 2

# bamfile or list
bamfn=$1
# check if we have single bam of list of bamlfiles as argument
if [ -e "$bamfn" -a -r "$bamfn" ]; then
    bamrel=$(basename -- "$bamfn")
    bamext="${bamrel##*.}"

    if [ "$bamext" = "txt" ]; then
        inputtype="--bam-list"
    elif [ "$bamext" = "bam" ]; then
        inputtype="--bam-file"
    else
        echo "ERROR: Please provide a .bam or .txt file."
        exit 1
    fi
else
    echo "ERROR: $bamfn does not exist or not readable."
    exit 1
fi

# create outdir
if ! [ -d "$outdir" -a -w "$outdir" ]; then
    mkdir -p "$outdir"/bcf
fi

if [ $? -ne 0 ]; then
    echo "ERROR: Creating $outdir failed. Please check if you have write access."
    exit 1
fi


# fix nomenclature
if [ "$ref" = "grch37" -o "$ref" = "GRCH37" -o "$ref" = "GRCh37" ]; then
    ref="GRCh37"
elif [ "$ref" = "grch38" -o "$ref" = "GRCH38" -o "$ref" = "GRCh38" ]; then
    ref="GRCh38"
elif [ "$ref" = "hg19" -o "$ref" = "Hg19" ]; then
    ref="hg19"
elif [ "$ref" = "hg38" -o "$ref" = "Hg38" ]; then
    ref="GRCh38"
else
    echo "ERROR: The reference must be GRCh37, GRCh38, hg19 or hg38"
    exit 1
fi

refdir="ref_$ref"
trans="trans_$ref.tsv"
sites="sites_$ref.txt"

if [ "$verbose" -eq 1 ]; then
    echo "[ $0 ]"
    if [ "$inputtype" = "--bam-list" ]; then
        echo "BAMLIST:   $bamfn"
    else
        echo "BAMFILE:   $bamfn"
    fi
    if [ "$ref" = "GRCh37" -o "$ref" = "hg19" ]; then
        echo "REFERENCE: $ref     (please ensure all BAM files are aligned to this reference)"
    else
        echo "REFERENCE: GRCh38/hg38"
    fi
    echo "THREADS:   $threads"
    echo "OUTDIR:    $outdir"
    echo "_____________________________________"
fi

# check if we look for parallel in our path
if [ "$parallel" -eq 1 ]; then
    PARALLEL=$( which parallel )
fi

if [ -x "$PARALLEL" ]; then
    # parallel phasing chunks
    # we have 11 chunks to impute, if threads are less<22 just use parallel to impute chunks parallelly
    # if we have more than nx11 threads, allocate more threads to GLIMPSE2_phase as well
    if [ "$threads" -lt 22 ]; then
        pthreads=$threads
        threads=1
    else
        pthreads=11
        threads=$(( threads/11 ))

    fi

    if [ "$verbose" -eq 1 ]; then
        echo "* Phasing samples parallelly with $pthreads * $threads threads. (this may take long)."
    fi

    ls "$DATA"/"$refdir"/*.bin | parallel -j "$pthreads" "$G2PHASE --threads $threads $inputtype $bamfn --reference {} --output $outdir/bcf/{/.}.bcf" >"$outdir"/phase.log

    if [ $? -ne 0 ]; then
        echo "ERROR: Phasing failed. Are all BAM files aligned to $ref""? Check $outdir/phase.log"
        exit 1
    fi
else
    # phase chunks sequentially
    for chunk in "$DATA"/"$refdir"/*.bin; do
        base=$(basename -- "$chunk")
        prefix="${base%.*}"
        outfn="$outdir/bcf/$prefix.bcf"

        if [ $verbose -eq 1 ]; then
            echo "* PHASING chunk $prefix (this may take long)."
        fi

        "$G2PHASE" --threads "$threads" "$inputtype" "$bamfn" --reference "$chunk" --output "$outfn" >"$outdir"/phase.log

        if [ $? -ne 0 ]; then
            echo "ERROR: Phasing failed. Are all BAM files aligned to $ref""? Check $outdir/phase.log"
            exit 1
        fi
    done
fi

# get list of imputed bcf files
ls "$outdir"/bcf/*.bcf >"$outdir"/BCFLIST.txt

if [ $? -ne 0 ]; then
    echo "ERROR: No phased bcf files? Are all BAM files aligned to $ref""? Check $outdir/phase.log"
    exit 1
fi

# ligate chunks
if [ $verbose -eq 1 ]; then
    echo "* LIGATING phased data."
fi

"$G2LIGATE" --threads "$threads" --input "$outdir"/BCFLIST.txt --output "$outdir"/imputed_variants.bcf >"$outdir"/ligate.log

if [ $? -ne 0 ]; then
    echo "ERROR: Ligation failed. Check $outdir/ligate.log"
    exit 1
fi

# query Hplex-S variants
if [ $verbose -eq 1 ]; then
    echo "* Filtering HirisPlex-S variants."
fi

"$BCFTOOLS" query -f '%CHROM\t%POS\t%REF\t%ALT[\t%SAMPLE=%GT]\n' -R "$DATA"/"$sites" "$outdir"/imputed_variants.bcf >"$outdir"/HISplex_variants.tsv

if [ $? -ne 0 ]; then
    echo "ERROR: bcftools query for HIrisPlex-S variants failed."
    exit 1
fi


# translate hplex query to HIrisPPlex-S csv format
if [ $verbose -eq 1 ]; then
    echo "* Converting $ref variants to HIrisPlex-S allele counts."
fi

"$BINDIR"/transToHISplex "$DATA"/"$trans" "$outdir"/HISplex_variants.tsv >"$outdir"/HISplex41_upload.csv
if [ $? -ne 0 ]; then
    echo "! Translating imputed genotypes to HIrisPlex-S format failed."
    exit 1
fi

if [ $verbose -eq 1 ]; then
    echo "* Output file: $outdir/HISplex41_upload.csv"
    echo "Done."
fi
# End of file

