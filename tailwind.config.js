module.exports = {
  mode: "jit",
  purge: ["./views/**/*.html"],
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
    require('@tailwindcss/aspect-ratio'),
  ],
};
