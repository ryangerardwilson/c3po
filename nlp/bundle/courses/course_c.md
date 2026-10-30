# C

## Part I: K&R 2nd Ed Concepts

### Section A: K&R Chapter 1

#### Lesson 1: Hello World

    #include <stdio.h>

    int main() {
        printf("Hello, world!");
        return 0;
    }

#### Lesson 2: Compile & Execute

    clang main.c -o main && ./main

#### Lesson 3: Basic Data Types and Their Qualifiers

    #include <stdio.h>

    int main() {

        // The 4 Complete Basic Data Types 
        // Complete means -> Compiler allocates memory 
        // based on type declaration
        int i = 1337;              
        float f = 3.14f;           
        double d = 3.1415926535;   
        char c = 'L';             
        // Equivalent:
        char c2 = 76;

        return 0;
    } 

#### Lesson 4: Derived Data Types

    #include <stdio.h>

    int main() {

        // Array: A contiguous block of the same complete type
        // Strings are arrays derived from chars, the stdlib assumes this set up
        char name[10] = "Linus";  // Ascii bytes: L,i,n,u,s,\0,\0,\0,\0,\0
        // Derived from int
        int numbers[5] = {1, 2, 3, 4, 5};
        // Derived from float
        float decimals[4] = {3.14, 2.718, 1.618, 0.577};

        return 0;
    } 

#### Lesson 5: Formatting with Printf

    #include <stdio.h>

    int main() {

        char name[10] = "Linus";
        int age = 55;
        float height = 5.11f;
        double pi = 3.14159;
        long net_worth = 1000000000000L;

        // %s for strings: null-terminated or segfault
        printf("Name: %s\n", name);
        // %d for decimal integers.
        printf("Age: %d\n", age);
        // Minimum width 3, pads with spaces. Useful for alignment
        printf("Width: %3d\n", age);
        // %f for floats, default 6 decimal places--ugly as sin
        printf("Height: %f\n", height);
        // Precision: 2 digits after decimal
        printf("Prec: %.2f\n", height);

        return 0;
    } 

#### Lesson 6: Basic Arithmetic Expressions

    #include <stdio.h>

    int main() {
        double result;
        result = 2 * (2 + 3) / 4.0 - 1;
        printf("%f\n", result);
        return 0;
    } 

#### Lesson 7: If-Else

    #include <stdio.h>

    int main() {
        int age = 25;
        if (age >= 18) {
            printf("Adult\n");
        } else {
            printf("Kid\n");
        }
        return 0;
    } 

#### Lesson 8: While Loop

    #include <stdio.h>

    int main() {
        char s[] = "hello";
        int i = 0;
        while (s[i] != '\0') {
            i++;
        }
        printf("Length: %d\n", i);
        return 0;
    } 

#### Lesson 9: For Loop - Repeat Without Being a Loop Idiot

    #include <stdio.h>

    int main() {
        int i;
        for (i = 0; i < 10; i++) {
            printf("%d\n", i * i);
        }
        return 0;
    } 

#### Lesson 10: Getchar, Putchar, EOF

    #include <stdio.h>

    int main() {
        int c;
        printf("Type text, Ctrl+D to flush buffer"); 
        printf("OR: Enter, Ctrl+D to trigger EOT, which c interprets as EOF"); 

        // NOTE: EOF is
        // 1. not a character, but a macro defined in stdio.h as -1 
        // 2. what getchar() spits back when there's no more input to read 
        while ((c = getchar()) != EOF) {
            putchar(c);
        }
        return 0;
    }

#### Lesson 11: scanf

    #include <stdio.h>

    int main() {
        char name[20];
        printf("Enter name: ");
        scanf("%s", name);
        printf("Hello, %s\n", name);
        return 0;
    } 

#### Lesson 12: Constants, Mutables, Globals & Functions

    #include <stdio.h>

    #define INC 1 // Preprocessor constant
    const int G_CONST = 42; // Global const

    int global = 0; // Mutable global: mostly bad

    int add(int a, int b) {
        const int L_CONST = 10; // Local const
        int sum = a + b + L_CONST;
        global += INC;
        return sum;
    }

    int main() {
        printf("%d\n", add(3, 4)); 
        {
            const int b_const = G_CONST; // Block const
            printf("Block: %d\n", b_const);
        }
        printf("Global: %d\n", global); // 1
        return 0;
    }

### Section B: K&R Chapter 2

#### Lesson 1: Variable Names

    #include <stdio.h>

    int main() {
        int valid_best_practice_name = 42; // Starts with letter; lower case;
                                           // uses underscores
        int _underscore_start = 43; // OK, but avoid double `__`, as it may 
                                    // conflict with lib methods
        int name_with_digits123 = 44; // Digits fine after first char
        int lower_case = 1; // Case matters
        int LOWER_CASE = 2; // Different from above
        return 0;
    }

#### Lesson 2A: Data Types and Sizes

    // There are only a few basic data types in C: 
    // char:    a single byte, capable of holding one character in the local 
    //          character set
    // int:     an integer, typically reflecting the natural size of integers 
    //          on the host machine
    // float:   single-precision floating point
    // double:  double-precision floating point

    // Integer Qualifiers
    short int sh; // atleast 16 bits
    long int counter; // atleast 32 bits

    // Integer/ Char Qualifiers
    signed int si; // default, even if signed qualifier is not specified
    unsigned int usi; // always positive, and holds 2x more positive integers
    signed char sc; // maps the 256 ASCII values to the int range -128 to 127
    unsigned char usc; // has ASCII decimal values from 0 to 255

    // Float/ Double Qualifiers
    long double ld; // extended-precision floating point    

#### Lesson 2B: Data Types and Sizes

    // limits.h and float.h contain symbolic constants for the sizes allocated
    // by types and their qualifiers
    #include <stdio.h>
    #include <limits.h>  // For INT_MAX etc.
    #include <float.h>   // For FLT_MAX etc.

    int main() {
        // %zu is the C99 specifier for printing size_t (unsigned, from 
        // sizeof); use it to avoid UB with %d on large values
        printf("char: %zu bytes\n", sizeof(char));  // 1, always
        printf("int: %zu bytes\n", sizeof(int));    // 4, probably
        printf("long: %zu bytes\n", sizeof(long));  // 8 on 64-bit
        printf("Unsigned int max: %u\n", UINT_MAX);
        printf("Float max: %e\n", FLT_MAX);     // Max float value, 
                                                // in scientific notation
        return 0;
    }

#### Lesson 3A: Constants & Qualifiers

    #include <stdio.h> 

    int main() {
        // Integer constants (literals): decimal by default, octal (0 prefix), 
        // hex (0x prefix)
        int dec = 42;   
        int oct = 052;  
        int hex = 0x2A; 

        return 0;
    }


#### Lesson 3B: Integer Constants with Suffixes

    #include <stdio.h>

    int main() {
        // Look, in C, integers are ints by default, but you can force them to be 
        // long or unsigned. Don't be sloppy; use suffixes to avoid surprises on 
        // different machines
        long big_num = 123456789L;  // L for long
        unsigned int positive_only = 42U;  // U for unsigned;
                                           // doubles positive range
        unsigned long even_bigger = 0xFUL; // Hex with UL suffix, value 15 decimal 
                                           // Hex is for humans who hate octal

        printf("Long: %ld, Unsigned: %u, Unsigned Long: %lu\n", big_num, positive_only, even_bigger);
        // Octal? Prefix with 0. Like 037 is 31 decimal. But seriously, use hex 
        // instead unless you're punishing yourself
        int octal = 037;
        printf("Octal 037 as decimal: %d\n", octal);

        return 0;
    }

#### Lesson 4: Floating-Point Constants

    #include <stdio.h>

    int main() {
        // Floats and doubles: decimals or exponents. Double by default
        double pi_approx = 3.14159;  // Decimal point makes it floating-point
        float small_pi = 3.14159F;   // F suffix for float; use when memory is 
                                     // tight, but don't cry about lost precision
        long double super_precise = 1.23456789e-10L;  // e for exponent, L for 
                                                      // long double. For when 
                                                      // double isn't enough

        printf("Double: %.5f, Float: %.5f, Long Double: %.10Lf\n", pi_approx, small_pi, super_precise);
        // Pro tip: Avoid floats unless you're optimizing arrays. Doubles are the 
        // way to go on modern hardware

        return 0;
    }

#### Lesson 5: Character Constants and Escape Sequences

    #include <stdio.h>

    int main() {
        // Characters are just ints in disguise. 'x' is the ASCII value of x, not the string "x".
        char letter = 'x';  // Single quotes. Value? Probably 120 in ASCII
        char null_char = '\0';  // Null terminator. Numeric value 0, but write it as char for clarity
        char bell = '\a';  // Escape for alert (beep). Try it; annoy your terminal
        char newline = '\n';  // Newline, obviously
        char tab = '\t';  // Horizontal tab

        // Octal escapes: '\ooo' for arbitrary bytes
        char vtab_oct = '\013';  // Vertical tab in octal
        // Hex escapes: '\xhh' for the same, but hex is saner
        char bell_hex = '\x7';  // Bell in hex

        printf("Char value: %d\n", letter);  // Prints the int value
        printf("Escape demo: Alert\a Newline\n Tab\t Done.\n");
        // Backslash itself? '\\'. Quotes? '\'' or '\"'. Don't mess this up or your strings break

        // Full escapes: \a \b \f \n \r \t \v \\ \? \' \" \ooo \xhh
        // Use them in strings too, like "Hello\nWorld"

        return 0;
    }

#### Lesson 6: String Constants and Concatenation

    #include <stdio.h>
    #include <string.h>  

    int main() {
        // Strings: Double quotes, null-terminated arrays under the hood
        char greeting[] = "Hello, world";  // Includes '\0' at end. Storage: length + 1
        char empty[] = "";  // Just '\0'

        // Concatenation at compile time: Split long strings
        char long_str[] = "This is a long string that "
                          "spans multiple lines. Neat, huh?";

        printf("%s\n", long_str);
        // Length? Use strlen, excludes '\0'
        printf("Length of greeting: %zu\n", strlen(greeting));  // 12, not 13

        // Don't confuse 'x' (int) with "x" (array with 'x' and '\0')
        // 'x' + 1 is 'y' maybe, but "x"[0] is 'x'

        return 0;
    }

#### Lesson 7: Enumeration Constants

    #include <stdio.h>

    int main() {
        // Enums: Named int constants. Better than #define for related values
        enum boolean { NO, YES };  // NO=0, YES=1 by default.
        // Starts at 1, increments
        enum months { JAN = 1, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC };  

        // Partial values: Gaps? Fine
        enum escapes { BELL = '\a', BACKSPACE = '\b', TAB = '\t', NEWLINE = '\n', VTAB = '\v', RETURN = '\r' };

        printf("YES: %d, FEB: %d, BELL: %d\n", YES, FEB, BELL);
        // Use in vars: enum boolean flag = YES;
        // Compilers might not enforce values, but debuggers love symbolic names
        // Alternative to #define, auto-generates values. Portable and readable

        // Distinct enums can't share names, but values can overlap

        return 0;
    }

#### Lesson 8: Constant Expressions

    #include <stdio.h>

    #define MAXLINE 1000  

    int main() {
        // Constants only, evaluated at compile time
        char line[MAXLINE + 1];  // Array size must be constant.

        // In arrays or wherever constants are needed
        #define LEAP 1
        int days[31 + 28 + LEAP + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 31];  // For leap year

        printf("Size of line: %zu\n", sizeof(line));  // 1001 bytes

        // Useful for portability; no runtime cost

        return 0;
    }

#### Lesson 9: Declarations and Initializations

    #include <stdio.h>

    #define MAXLINE 1000  

    int main() {
        // 1. Declare types and variables before use
        int lower, upper = 100, step;  // Multiple, some initialized.
        char c = 'a';
        char line[1000] = {0};  // Zero-init array

        // 2. Use = to Init
        int i = 0;
        int limit = MAXLINE + 1;  // From #define
        float eps = 1.0e-5;
        printf("Upper: %d, Eps: %e\n", upper, eps);

        return 0;
    }

#### Lesson 10: The Const Qualifier

    #include <stdio.h>
    #include <string.h>

    int main() {
        // Const: Can't change. For safety
        const double e = 2.71828182845905;  // Try e = 3; compiler yells
        const char msg[] = "warning: ";  // Array elements immutable

        // In functions: Promises no change
        // int strlen(const char[]);  // From string.h
        printf("%s %f\n", msg, e);

        return 0;
    }

#### Lesson 11: Arithmetic Operators

    #include <stdio.h>

    int main() {
        // +, -, *, /, % (modulus).
        int x = 10, y = 3;
        printf("Add: %d, Sub: %d, Mul: %d\n", x + y, x - y, x * y);
        printf("Div: %d (truncates), Mod: %d\n", x / y, x % y);  // 3 and 1

        // % only for ints. No floats
        // Negative? Machine-dependent. Avoid if portable

        // Precedence: * / % > + -. Left to right
        int res = 1 + 2 * 3;  // 7, not 9

        return 0;
    }

#### Lesson 12: Relational and Logical Operators

    #include <stdio.h>

    int main() {
        // Relational: > >= < <=. Same precedence
        // Equality: == !=. Lower than relational
        // All lower than arithmetic
        int i = 5, lim = 10;
        if (i < lim - 1) printf("Yes.\n");  // i < (lim-1)

        // Logical: && ||. Short-circuit: Stops early
        // && > || precedence. Lower than relational
        int c = 'a';
        if (i < lim - 1 && (c = getchar()) != '\n' && c != EOF) {}  // From getline

        // No parens needed for && chain
        // But for assignment: (c = getchar()) != '\n'

        // True=1, False=0
        int valid = 1;
        if (!valid) printf("Not valid.\n");  // ! negates: Non-zero to 0

        return 0;
    }

## Part II: Scripting

### Section A: Foundational Patterns

#### Lesson 1: Stdin/Files Multi Input Loop 

    #include <stdio.h>
    #include <stdlib.h>

    // Pattern: most common CLI skeleton -> default stdin, or process each file arg
    // Teaches: argc/argv basics, FILE*, error paths, clean shutdown

    void process(FILE *f) {
        int ch;
        while ((ch = fgetc(f)) != EOF)
            putchar(ch);                    // echo for demo
    }

    int main(int argc, char **argv) {
        FILE *f = stdin;

        if (argc == 1) {                    // no args -> stdin
            process(f);
        } else {
            for (int i = 1; i < argc; i++) {
                f = fopen(argv[i], "r");
                if (!f) {
                    fprintf(stderr, "%s: ", argv[0]);
                    perror(argv[i]);
                    return 1;
                }
                process(f);
                fclose(f);
            }
        }
        return 0;
    }

#### Lesson 2: Safe getline loop

    #include <stdio.h>
    #include <stdlib.h>

    // Pattern: modern, safe line reading -> no buffer overflow risk
    // Teaches: dynamic allocation, ownership (must free), ssize_t

    int main(void) {
        char *line = NULL;
        size_t cap = 0;
        ssize_t len;

        while ((len = getline(&line, &cap, stdin)) != -1) {
            // len includes \n if present
            if (len > 0 && line[len-1] == '\n')
                line[--len] = '\0';         // trim newline for clean use

            printf("[%zd] %s\n", len, line);
        }

        free(line);                         // always free what getline gave us
        return 0;
    }

#### Lesson 3: Simple Flag Option Parser

    #include <stdio.h>
    #include <string.h>
    #include <stdlib.h>

    // Pattern: manual argv scan -> learn before getopt
    // Teaches: string compare, flag accumulation, usage guard

    int main(int argc, char **argv) {
        int verbose = 0;

        for (int i = 1; i < argc; i++) {
            if (!strcmp(argv[i], "-v") || !strcmp(argv[i], "--verbose"))
                verbose++;
            else if (argv[i][0] == '-') {
                fprintf(stderr, "Unknown option: %s\n", argv[i]);
                return 1;
            } else {
                printf("Processing: %s (verbose=%d)\n", argv[i], verbose);
            }
        }
        return 0;
    }

#### Lesson 4: Error Reporting with Context

    #include <stdio.h>
    #include <stdlib.h>
    #include <string.h>

    // Pattern: consistent, helpful error messages -> makes your tools debuggable
    // Teaches: stderr vs stdout, perror usage, argv[0] as program name, exit(1)

    int main(int argc, char **argv) {
        if (argc < 2) {
            // Use stderr for errors + include program name -> scripts can distinguish
            fprintf(stderr, "%s: missing filename\n", argv[0]);
            fprintf(stderr, "Usage: %s file.txt\n", argv[0]);
            return 1;
        }

        FILE *f = fopen(argv[1], "r");
        if (!f) {
            // perror adds system error string (e.g. "No such file or directory")
            fprintf(stderr, "%s: cannot open '%s': ", argv[0], argv[1]);
            perror("");                     // prints the errno message
            return 1;
        }

        // Normal output goes to stdout
        int ch;
        while ((ch = fgetc(f)) != EOF)
            putchar(ch);

        fclose(f);
        return 0;
    }

#### Lesson 5: State Machine for Token Scanning

    #include <stdio.h>
    #include <ctype.h>

    // Pattern: state machine for word counting -> single pass, no extra memory
    // Teaches: getchar(), EOF handling, state transitions, isspace()

    int main(void) {
        int c;
        int in_word = 0;
        long words = 0;
        long chars = 0;

        fputs("Type text (finish with Ctrl+D):\n", stderr);  // hint to user

        while ((c = getchar()) != EOF) {
            chars++;

            if (isspace(c)) {
                in_word = 0;
            } else {
                if (in_word == 0) {
                    words++;
                    in_word = 1;
                }
            }
        }

        // After Ctrl+D, we reach here
        printf("\n%ld words, %ld characters\n", words, chars);
        return 0;
    }

#### Lesson 6: Dynamic String Builder 

    #include <stdio.h>
    #include <stdlib.h>
    #include <string.h>

    // Pattern: build string incrementally -> no fixed-size worry
    // Teaches: realloc geometric growth, ownership, manual null-termination

    typedef struct {
        char  *data;
        size_t len;
        size_t cap;
    } StrBuf;

    void sb_append(StrBuf *sb, const char *s) {
        size_t n = strlen(s);
        if (sb->len + n + 1 > sb->cap) {           // +1 for null byte
            sb->cap = sb->cap ? sb->cap * 2 : 64;   // double capacity (amortized O(1))
            sb->data = realloc(sb->data, sb->cap);
            if (!sb->data) abort();
        }
        memcpy(sb->data + sb->len, s, n);
        sb->len += n;
        sb->data[sb->len] = '\0';
    }

    int main(void) {
        StrBuf sb = {0};

        sb_append(&sb, "Hello");
        sb_append(&sb, ", ");
        sb_append(&sb, "world from dynamic C string!\n");

        fputs(sb.data, stdout);

        free(sb.data);                      // only one free needed
        return 0;
    }
