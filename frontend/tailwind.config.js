/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html','./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#eef6ff',
          100: '#d9ebff',
          200: '#b9d9ff',
          300: '#8ec0ff',
          400: '#5ca1ff',
          500: '#2d7dff',
          600: '#155ef0',
          700: '#1449c2',
          800: '#133e9b',
          900: '#12367d',
        }
      },
      boxShadow: {
        'soft': '0 10px 30px -12px rgba(2, 55, 140, 0.25)',
        'inset': 'inset 0 1px 0 0 rgba(255,255,255,.3)'
      }
    },
  },
  plugins: [],
}
