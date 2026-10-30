# Unix

## Part I: Typing in the Terminal

### Lesson 1: Multiline Strings

    # Option A: Use `fc`. It loads the last command into vim, and executes when
    # you :wq

    # Option B: Use `\`. The `\` is ignored, and the below is executed as if it
    # were one line
    bash$ p\
    >w\
    >d
    /home/ryan/Apps/rtutor/courses

    # NOTE: The below will not print 'hello world' in one line because single 
    # quotes tell bash to treat everything inside '' as a string literal
    bash$ echo 'hello\
    > world'

    # Insted, try this:
    bash$ echo \
    > 'hello world'

### Lesson 2: Cursor Control

    Ctrl+A: Move cursor to the beginning of the line
    Ctrl+E: Move cursor to the endo of the end
    Alt+F: Move cursor forward by one word
    Alt+B: Move cursor backwards by one word

### Lesson 3: Text Control

    Ctrl+U: Clear the line before the cursor
    Ctrl+K: Clear the line after the cursor

    Ctrl+W: Remove one word to the left
    Alt+D: Remove one word to the right

    Ctrl+H: Remove one char to the left
    Ctrl+D: Renive one char to the right
    Tab: Auto-complete

### Lesson 4: Job Control

    Ctrl+C: Abort a foreground job by sending a SIGINT signal
    Ctrl+Z: Stop a foreground job by sending a SIGSTOP signal

### Lesson 5: Session Control

    Ctrl+L: Clear current terminal session
    Ctrl+D: Exit the current terminal session

### Lesson 6: History Search

    Ctrl+R: Reverse search. Keep hitting Ctrl+R to cycle through older matches;
            hit tab to select but not execute; hit enter to select and execute
    Ctrl+P: Previous command
    Ctrl+N: Next command
    !!: Run the last command again

## Part II: Vim CLI Tools

### Lesson 1: Toggle vim normal mode in Alacrity

    # This is a toggle, use the same combo to turn off vim normal mode
    Ctrl+Shift+Space

### Lesson 2: Use vim Buffers

    vim file1 file2 # Open specific files as buffers
    vim dir/* # Open all files as buffers
    vim dir/*.md # Open files of specific extensions as buffers
    vim dir/**/*.md # Same as above, but recursively. 
                    # The dir should have subdirs

## Part III: Tmux

### Lesson 1: Sessions

    tmux new -s dev # Start
    Ctrl-b d  # Deteach 
    tmux ls  # List 
    tmux attach -t dev  # Jump back in

### Lesson 2: Panes

    Ctrl-b %  # Side-by-side
    Ctrl-b "  # Top-bottom panes.
    Ctrl-b arrow/o   # Switch panes.
    Ctrl-b x  # Kill current pane (confirm with y).

### Lesson 3:  Copy Mode

    Ctrl-b [  # Enter copy mode 
    arrow/hjkl # Navigate 
    q # exit
    Space # Initiate selection Mode
    ESc # Quit selection mode 

## Part IV: Search Utils

### Lesson 1: rg (ripgrep) 

    # Search file globs recursively from the current dir
    rg --files course*.md
    #! course_unix.md
    #! course_math.md
    #! course_sql.md
    #! course_c.md
    #! course_python.md
    #! course_a.md

    # Search files for specific content
    rg "python"
    #! main.py
    #! 1:#!/usr/bin/env python3

### Lesson 2: ls

    # Search files starting with 'build', having specific extension
    ls build*.c build*.h
    # Search files not starting with 'build', have specific extension
    ls !(build*).c
    # Search files not starting with 'foo' or 'bar', but having specific extension
    ls !(@(foo|bar)*).c

### Lesson 3: Grep, Find & Locate

    # Grep searches text within files
    grep "error" system.log
    grep -r "error" logs/ # Search dirs.

    # Find searches file names
    find /project -name "*.py" # Match names/extensions.
    find . -type f -size +50M # Files >50MB.
    find /backup -mtime -7 # Last 7 days.

    # Xargs combo to find strings across multiple files of a specific extension
    find . -type f -name "*.py" | xargs grep "import pandas as pd"

    # Locate: While find seaches below ~/, locate seaches above 
    sudo updatedb && locate "nginx"

### Lesson 4: Diff and Git Diff

    # Diff compares two files line by line
    diff file1.txt file2.txt  
    diff -u file1.txt file2.txt  # Unified format, easier to read 
    diff -r dir1/ dir2/  # Recursively compare directories.

    # Git diff: 
    git diff  # Shows changes between working dir and index.
    git diff --cached  # Changes staged for commit.
    git diff HEAD  # All changes since last commit.
    git diff branch1 branch2  # Compare two branches.

## Part V: File & Directory Management

### Lesson 1: File & Dir CRUD

    # Create
    touch newfile.txt  
    mkdir newdir  
    mkdir -m 755 secure_dir  

    # Read
    ls
    ls -alh # All details
    ls -lh # Exclude hidden files
    ls -Rlh # Recursive: List subdirectories too
    ls -1 # Display has vertical list
    ls -t  # Sort by modification time (newest first)
    ls -S  # Sort by size (largest first)
    ls *.txt  # Wildcard: List only text files
    ls -l | grep "^d"  # Pipe to grep: Show only directories

    # Update
    cp file.txt backup/  
    cp -r dir1/ dir2/  
    mv oldname.txt newname.txt  
    mv file.txt /new/path/  
    chmod 644 script.sh  # Set octal 
    chmod -R 644 dir/  # Recursive apply
    chmod +x script.sh # Make executable
    # 600 for private files; 644 for standard files; 700 for private dirs; 
    # 755 for letting others peek/run without editing; 777 for all access
    # In unix systems, inside the /home dir, for newly created:
    # - files:  644 is default because it strikes a balance between you owning 
    #           the file, the others accessing it without editing.
    # - dirs:   755 is default because it allows others to cd into your dirs as 
    #           the cd command needs executable permission

    # Delete
    rm file.txt  
    rm *.py
    rm -rf dir/  

### Lesson 2: Navigating with cd and pwd

    pwd  
    cd  
    cd /path
    cd ../..  # Up two levels
    cd ~  
    cd -  # Back to previous directory

## Part VI: CTL Utilities

### Lesson 1: Why ctl utilities are stateful

    # ctl-style tools (iwctl, bluetoothctl, etc.) are interactive clients for
    # long-running daemons, not one-shot commands.
    
    # Why they feel different from simple CLI tools:
    # - system state changes continuously (scan results, device availability)
    # - operations are async (pairing/auth/discovery can take time)
    # - you often re-run list/show commands to refresh live state
    
    # Mental model:
    # simple CLI -> request/response/exit
    # ctl utility -> attach/session/events/state refresh

### Lesson 2: iwctl

    iwctl

    # Inside the iwctl cli
    device list
    station wlan0 scan
    station wlan0 get-networks
    station wlan0 connect "Network Name"
    # To disconnect
    station wlan0 disconnect
    # To view iwctl commands
    help
    # To exit iwctl cli
    exit

### Lesson 3: bluetoothctl

    bluetoothctl
    power on
    agent on
    default-agent
    scan on

    # Put target device in pairing mode, then:
    pair <MAC>
    trust <MAC>
    connect <MAC>

    # Verify and inspect:
    devices
    info <MAC>

    # `default-agent` does NOT mean "default bluetooth device".
    # It only selects which agent handles pairing prompts.
    
    # You can pair/trust/connect multiple devices:
    pair <MAC_1>
    trust <MAC_1>
    connect <MAC_1>
    pair <MAC_2>
    trust <MAC_2>
    connect <MAC_2>
    devices

    # Optional cleanup:
    disconnect <MAC>
    remove <MAC>
    exit

## Part VII: Media Utils

### Lesson 1: Figlet

    figlet "Hello, World"
    figlet "Hello, World" > out.txt

    # List all fonts
    figlist

    # Invoke a specific font
    figlet -f smslant "Hello, World"

    # Use magick to convert to png
    # - point size governs how HD it is
    # - fill governs the font color
    # - backgroung governs the background color
    magick -font DejaVu-Sans-Mono -pointsize 50 -background black -fill white label:@t.txt t.png
