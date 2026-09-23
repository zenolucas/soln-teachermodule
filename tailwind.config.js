/** @type {import('tailwindcss').Config} */
module.exports = {
  // public/js/**/*.js is listed on its own, not folded into a blanket "./**/*.js" -
  // Tailwind's content globs aren't node_modules-aware like a bundler's would be, so a
  // broad "./**/*.js" would also crawl every installed package under node_modules
  // (see FE-05, which is what first needed a class - toast/alert - used only from a
  // hand-written script rather than a .templ/.go file).
  content: [ "./**/*.html", "./**/*.templ", "./**/*.go", "./public/js/**/*.js", ],
 safelist: [],
 plugins: [require("daisyui")],
 daisyui: {
   themes: ["nord"]
 }
}