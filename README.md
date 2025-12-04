aHISplex : Imputation based HIrisiPlex-S phenotyping from low coverage ancient (or degraded forensic) genome data.
======================================================
This is a repository containing the tools to predict the visible traits (hair, eye and skin colors) in humans from ancient/low coverage shotgun WGS DNA. The approach is described detail in our manuscript:
>Zoltán Maróti, Emil Nyerki, Endre Neparaczki, Tibor Török, Gergely István Varga, Tibor Kalmár: **aHISplex: an imputation-based method for eye, hair and skin colour prediction from low-coverage ancient DNA.**
>
>doi: (under revision)

The repository contains the golang tools to translate the reference based genotype information to the HIrisPlex-S system allele count based marker format, the tool to translate the HIrisPlex-S system phenotype probabilities output to the text based classification using the scheme described in the HirisPlex-S system User manual (https://hirisplex.erasmusmc.nl/pdf/hirisplex.erasmusmc.nl.pdf). And the script to glue the required public and own tools to carry out the imputation steps to the HIrisPlex-S web interface input file.

### Dependencies
The code depends on golang. Gloang version 1.18 was tested, but it should work on any higher golang versions as it uses only standard golang agnostic code. The framework also depends on external software for performing imputation (**GLIMPSE2**) and manipulation of VCF/BCF files (**bcftools**). These tools have to be installed and be executable (in the PATH environmental variable). **NOTE** that GLIMPSE2 software compiled with different BOOST library versions are binary incompatible when loading the GLIMPSE2 internal '.bin' reference files. Accordingly, we recommend using the statically compiled and frozen GLIMPSE 2.0.0 software available at Github: [GLIMPSE2_phase_static](https://github.com/odelaneau/GLIMPSE/releases/download/v2.0.0/GLIMPSE2_phase_static) and [GLIMPSE2_ligate_static](https://github.com/odelaneau/GLIMPSE/releases/download/v2.0.0/GLIMPSE2_ligate_static) to ensure that the included precompiled GLIMPSE2 reference binary files are readable. If only dynamically linked GLIMPSE2 is available it may be necessary to create the .'bin' files manually (based on the GLIMPSE2 tutorial to prepare reference binary data) and replace the included reference files in the **aHISplex_data/** directory.

## optional dependencies
In case GNU parallel is installed on the system, then the imputation of the 11 different genome regions (harbouring the 41 HirisPlex-S markers) can be run parallel.

### Installation and Building
To install and build, ensure you have `go` and `make` installed. Clone (or `go get`) this repo into your `$GOPATH`:
```sh
go get github.com/zmaroti/aHISplex
```

Enter the package directory and type
```sh
make
```
to build the command line tools, and install binaries and the precompiled reference data in the default user's local **~/bin/** directory. If the default install directory is unavailable, the Makefile has to be manually edited to change the *INST_DIR* variable to the place of installation.


### Usage
The main shell script has the following options.
```
aHISplex.sh [-h?ns] [-t threads] [-o OUTPREF|DEFAULT:impute] [-r reference|DEFAULT:GRCh37] <BAMFILE.bam|BAMLIST.txt>

Runs HIrisPlex-S analysis on a single bam file or a list of bam files. The markers are
genotyped by GLIMPSE2 using imputation based on the appropriate 1KG Phase III reference data set.
The 41 markers of the HirisPlex-S are filtered by bcftools. And the variants are translated
to the HIrisPlex-S allele count format by a custom tool.

optional arguments:
     -s             Silent mode, suppress verbose information on the analysis.
     -n             Don't use parallel execution  even if parallel tool is available.
     -t threads     try to use more threads (default: 1)
     -o outputdir   the output directory of the imputed data (default: impute)
     -r reference   the reference the BAM files are mapped to. NOTE you can't mix GRCh37 and hg19
                    BAM files due to differences in chromosome naming convention (1 vs chr1).
                    This is not an issue in case of GRCH38 vs hg38 BAM files.
```
## additional tools to aid conversion and classification
The individual command line tools (classifHISplex, transToHISplex) will print a help if issued without the proper options or with the '-help' flag.

```sh
classifHISplex -help
transToHISplex -help
```

### Quickstart with practical examples
**Before analysis, please read the manuscript and understand the limitations of the method.** Based on the genome coverage, date of sample, the rate of PMD, the imputation, and the subsequent phenotype prediction may have significantly different accuracy. Furthermore, please **NOTE** that imputation relies on linked genomic context, hence our method can be applied to SHOTGUN WGS data only and was NOT tested on 1240K capture data, which likely have limited linked marker context harbouring the HIrisPlex-S markers.

The analysis consists of 3 steps.
1. Run aHISplex.sh
2. Upload the output of step 1 (HISplex41_upload.csv) to https://hirisplex.erasmusmc.nl/ web service. Download/save the resulting phenotype probability output file.
3. Use classifHISplex to evaluate phenotype probabilities applying the rules described in the HIrisPlex-s user manual.

The tool can analyse a single BAM file or a list of BAM files. During the first step, the data will be imputed and the result of the analysis is saved in the OUTPREF directory (as set by the -o option, or the "impute" directory if not provided). The directory will contain multiple files, including the log files of GLIMPSE2 imputation and ligation, the imputed genotypes of the 11 genome regions harbouring the 41 HirisPlex-S markers, the filtered genotypes of the 41 markers in bcf and also a .tsv format (from bcf query), and the upload file 'HISplex41_upload.csv' that contains the translated HIrisPlex-S allele counts to be uploaded on the HirisPlex-S system's web service (https://hirisplex.erasmusmc.nl/). This comma separated ('.csv') upload file may contain data for single sample or multiple samples (in case a BAM_LIST file is provided).
 
## analysing a single GRCh37 aligned BAM file
step 1:
```sh
aHISplex.sh bamfile.bam -o bamfile
```

step 2:
Upload the HISplex41_upload.csv (from bamfile subdirectory) to https://hirisplex.erasmusmc.nl/ web service, and download/save the resulting phenotype probability output file.

step 3:
Classify the phenotype probabilities as hair, eye, and skin phenotypes.
```sh
classifHISplex -short HIrisPlex-S_result.csv >classifications_short.csv

or

classifHISplex -short HIrisPlex-S_result.csv >classifications.csv
```

With the '-short' option, only the sample ID, the eye, hair, and skin phenotypes are saved in the '.csv' file. Without this option, all of the genotype and phenotype probability data is saved, and the last 3 columns will contain the phenotype classifications, according to the classification schema described in detail in the HirisPlex-S user manual.

## analysing multiple samples in a single run
step 1:
```sh
aHISplex.sh BAMLIST.txt -o bamfile
```

step 2:
Upload the HISplex41_upload.csv to https://hirisplex.erasmusmc.nl/ web service, and download/save the resulting phenotype probability putput file.

step 3:
Classify the phenotype probabilities as hair, eye, and skin phenotypes.
```sh
classifHISplex -short HIrisPlex-S_result.csv >classifications_short.csv

or

classifHISplex -short HIrisPlex-S_result.csv >classifications.csv
```

**NOTE** the BAMLIST.txt is a standard GLIMPSE2 BAMLIST.txt file. One BAM file per line. Optionally, a second column (space separated) can be used to specify the sample name; otherwise the name of the file is used.

## analysing non GRCh37 based alignments
The tool included precompiled binaries for GRCh37, hg19, GRCh38, and hg38. With the '-r' option this can be specified at step 1. **NOTE** GLIMPSE2_phase cannot deal with the different naming conventions of GRCH37/hg19 (N vs chrN) and can only be run when the alignments and the reference data have the same naming conventions. Accordingly, alignments (BAM files) with different references have to be analyzed separately.
```sh
aHISplex.sh -r hg38 hg38_aligned_BAM_file.bam
```

## optimizing parallel analysis
In case GNU parallel is installed, the software tries to use all available CPU cores to parallelly run the imputation of the 11 genome regions. The number of cores can be overridden by the '-t' option of aHISplex.sh. Since GLIMPSE2_phase can also use more threads to impute a chunk of genome region, it is recommended to specify core numbers that are divisible by 11 (the number of genome chunks harboring the 41 HirisPlex-S system markers). Depending on whether your architecture has multithreading or not, you can provide higher than available physical or virtual CPU cores, as the Linux OS will then split the available system resources between threads. For example, on a 20-virtual-core machine, you can specify 22 cores (*-t 22*) for optimal speed analysis, allowing parallel imputation of the 11 genome regions with 2 threads allocated to each GLIMPSE2_phase process. However, it is not recommended to significantly overcommit the available CPU cores, as it may lead to too much content switching between tasks and underwhelming performance. Since the imputation is only performed on the regions harbouring the 41 HIrisPlex-S markers (with a 5mb flanking genomic context), the analysis should be relatively quick compared to whole-genome imputations.
