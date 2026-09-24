/** @type {import('tailwindcss').Config} */
module.exports = {
  // public/js/**/*.js is listed on its own, not folded into a blanket "./**/*.js" -
  // Tailwind's content globs aren't node_modules-aware like a bundler's would be, so a
  // broad "./**/*.js" would also crawl every installed package under node_modules
  // (see FE-05, which is what first needed a class - toast/alert - used only from a
  // hand-written script rather than a .templ/.go file).
  content: [ "./**/*.html", "./**/*.templ", "./**/*.go", "./public/js/**/*.js", ],
 safelist: [],
 theme: {
   extend: {
     colors: {
       // Not a nord theme token - the revamp's design spec (01 Tokens) calls for this
       // one extra color outside daisyUI's "nord" palette (see DEC-1/DEC-12).
       polar1: "#3B4252",
     },
   },
 },
 plugins: [require("daisyui")],
 daisyui: {
   themes: ["nord"]
 }
}