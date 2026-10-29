/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  safelist: [{ pattern: /^(chip|alert)-(ok|bad|warn|muted|neutral|error)$/ }, 'logo-light', 'has-error', 'active', 'flush'],
  theme: {
    extend: {
      colors: {
        ink: '#0B1F3A',
        brand: { DEFAULT: '#0F766E', dark: '#0B5F58', soft: '#E6F4F2', glow: '#2DD4BF' },
        page: '#F4F6F9',
        line: '#E3E8EF',
        muted: '#5B6B82',
        danger: { DEFAULT: '#B42318', soft: '#FDECEA' },
        warn: { DEFAULT: '#B54708', soft: '#FEF4E6' },
        ok: { DEFAULT: '#067647', soft: '#E7F5EE' },
      },
      fontFamily: { sans: ['Inter', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'] },
      boxShadow: { card: '0 1px 2px rgba(11,31,58,.05)', pop: '0 12px 32px rgba(11,31,58,.18)' },
    },
  },
}
