import type { SVGProps } from 'react';

type P = SVGProps<SVGSVGElement> & { size?: number };
const base = (size: number, strokeWidth = 1.6) => ({
  width: size,
  height: size,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
});

export const SearchIcon = ({ size = 20, ...p }: P) => (
  <svg {...base(size, 2)} {...p}>
    <circle cx="11" cy="11" r="7" />
    <path d="M20 20l-4-4" />
  </svg>
);
export const UserIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <circle cx="12" cy="8" r="4" />
    <path d="M4 21c1.5-4 4.5-6 8-6s6.5 2 8 6" />
  </svg>
);
export const HeartIcon = ({ size = 22, filled = false, ...p }: P & { filled?: boolean }) => (
  <svg {...base(size)} fill={filled ? 'currentColor' : 'none'} {...p}>
    <path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10z" />
  </svg>
);
export const BagIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <path d="M5 8h14l-1 12H6L5 8z" />
    <path d="M9 8V6a3 3 0 0 1 6 0v2" />
  </svg>
);
export const TruckIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <path d="M3 7h11v9H3zM14 10h4l3 3v3h-7" />
    <circle cx="7" cy="18" r="1.6" />
    <circle cx="17" cy="18" r="1.6" />
  </svg>
);
export const ClockIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <circle cx="12" cy="12" r="8" />
    <path d="M12 8v4l3 2" />
  </svg>
);
export const GiftIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <path d="M4 9h16v11H4zM2 6h20v3H2zM12 6v14" />
  </svg>
);
export const LockIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <rect x="4" y="10" width="16" height="10" rx="2" />
    <path d="M8 10V7a4 4 0 0 1 8 0v3" />
  </svg>
);
export const ReturnIcon = ({ size = 20, ...p }: P) => (
  <svg {...base(size)} {...p}>
    <path d="M4 12a8 8 0 1 0 3-6.2M4 4v4h4" />
  </svg>
);
export const CheckIcon = ({ size = 34, ...p }: P) => (
  <svg {...base(size, 2)} {...p}>
    <path d="M5 12l5 5 9-10" />
  </svg>
);
export const MenuIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size, 1.8)} {...p}>
    <path d="M4 7h16M4 12h16M4 17h16" />
  </svg>
);
export const CloseIcon = ({ size = 22, ...p }: P) => (
  <svg {...base(size, 1.8)} {...p}>
    <path d="M6 6l12 12M18 6L6 18" />
  </svg>
);
