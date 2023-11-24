/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./views/**/*.html", "../scum/components/*.html"],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
    require('@tailwindcss/aspect-ratio'),
  ],
}
