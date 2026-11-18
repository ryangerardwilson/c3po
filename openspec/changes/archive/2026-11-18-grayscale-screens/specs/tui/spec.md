## ADDED Requirements

### Requirement: Screen paint is black, white, or gray
A screen SHALL paint text in black, white, or gray only. Red, green, and blue channels of one painted color SHALL be equal. A magnitude, such as a text file's line count or a binary's size, SHALL use a darker gray for a smaller value and a lighter gray for a larger value. Bold, dim, and reverse MAY mark a row or a state. A screen SHALL NOT paint cyan, yellow, red, green, or any other hue. `NO_COLOR` and a color mode of never SHALL leave that text unpainted.

#### Scenario: The directory tree scales in gray
- **WHEN** `r2d2 l` paints a text file and a binary with color on
- **THEN** each label SHALL use equal red, green, and blue channels
- **AND** a directory SHALL be white
- **AND** the tree glyphs SHALL be gray

#### Scenario: The spreadsheet cursor is gray
- **WHEN** the spreadsheet paints the focused cell with color on
- **THEN** the cell shade SHALL be black on light gray
- **AND** the cursor SHALL be white on a darker gray
- **AND** the frame SHALL NOT use an unequal red, green, and blue channel
