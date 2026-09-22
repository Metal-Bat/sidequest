# Sidequest visual prototype

`index.html` is a static character-sheet preview with example quests and an admin
account. Its forms and navigation require the live app. Live templates are in
`src/templates/`. Preview the design by serving this directory locally; its
stylesheet uses relative asset paths. The runtime embeds the stylesheet, selected
assets and local HTMX, excluding this README and the prototype HTML.

HTMX 2.0.10 is vendored in `vendor/htmx.min.js`, with its original license in
`vendor/LICENSE-HTMX.txt`. No CDN or frontend build step is needed at runtime.

Cinzel and Source Sans 3 are bundled in `fonts/` with their SIL Open Font License
files. Both font files come from the Google Fonts repository and are embedded in
the application. Headings use Cinzel; labels, forms, and body text use Source Sans 3.
