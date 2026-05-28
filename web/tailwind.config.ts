import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        // Premier League brand palette — deep purple as accent, used sparingly.
        'pl-purple': {
          50: '#faf5ff',
          100: '#f3e8ff',
          600: '#7e22ce',
          700: '#6b21a8',
          800: '#581c87',
        },
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'system-ui',
          'sans-serif',
        ],
      },
    },
  },
  plugins: [],
} satisfies Config
