# Fold the nlp trainer into a screen

`nlp.Run` is the screen entry. It follows the other screens: help, version, and a pipe with no resume id do not open a session. A terminal open stores a session whose path is the course slug and whose cursor is the menu position.

The course package keeps its own sqlx database name, `nlp`. Bundled course refresh uses the c3po version string. An edited course stays unbundled, so that refresh does not replace it.
