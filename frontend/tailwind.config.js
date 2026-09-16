/** @type {import('tailwindcss').Config} */
const { addDynamicIconSelectors } = require("@iconify/tailwind");

// Les couleurs sont declarees comme triplets RGB dans main.css (:root et .dark).
// Passer par des variables plutot que par des classes evite de doubler chaque
// regle d'une variante `dark:`, et garde un seul endroit ou lire la palette.
const jeton = (nom) => `rgb(var(${nom}) / <alpha-value>)`;

module.exports = {
  content: ["../views/**/*.{html,js,ts,go,templ}", "./src/**/*.{html,js,ts}"],
  darkMode: "selector",
  theme: {
    extend: {
      colors: {
        ground: jeton("--c-ground"),
        surface: jeton("--c-surface"),
        "surface-2": jeton("--c-surface-2"),
        ink: jeton("--c-ink"),
        "ink-2": jeton("--c-ink-2"),
        "ink-3": jeton("--c-ink-3"),
        line: jeton("--c-line"),
        "line-fort": jeton("--c-line-fort"),
        accent: jeton("--c-accent"),
        "accent-soft": jeton("--c-accent-soft"),
        "accent-ink": jeton("--c-accent-ink"),
        pine: jeton("--c-pine"),
        danger: jeton("--c-danger"),
        "danger-soft": jeton("--c-danger-soft"),
      },
      fontFamily: {
        display: ['"Archivo"', '"Helvetica Neue"', "Arial", "sans-serif"],
        sans: ['"Public Sans"', '"Helvetica Neue"', "Arial", "sans-serif"],
      },
      letterSpacing: {
        etiquette: "0.08em",
      },
      borderRadius: {
        // Un seul rayon, discret : ce sont les filets et les fonds qui separent
        // les blocs, plus les coins arrondis empiles.
        DEFAULT: "3px",
        md: "3px",
        lg: "4px",
        xl: "4px",
        "2xl": "5px",
        "3xl": "6px",
      },
      maxWidth: {
        lecture: "65ch",
      },
    },
  },
  plugins: [
    // Iconify plugin
    addDynamicIconSelectors(),
  ],
  safelist: [
    {
      pattern: /text-(green|rose)-500/,
    },
    "md:table-cell",
    "lg:table-cell",
    "xl:table-cell",
  ],
  variants: {
    customPlugin: ["responsive", "hover"],
  },
};
