/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#090d16',
        surface: '#111827',
        'surface-border': '#1f293d',
        accent: '#06b6d4',
        'accent-hover': '#0891b2',
      },
    },
  },
  plugins: [],
}
