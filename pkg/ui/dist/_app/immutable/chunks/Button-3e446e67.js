import{S as r,i as m,s as h,F as c,l as p,m as b,n as d,h as f,p as u,K as _,b as y,G as z,H as g,I as v,f as B,t as j}from"./index-e806b903.js";import"./theme-7d2f8787.js";function w(a){let s,i,n;const o=a[3].default,e=c(o,a,a[2],null);return{c(){s=p("button"),e&&e.c(),this.h()},l(t){s=b(t,"BUTTON",{type:!0,class:!0});var l=d(s);e&&e.l(l),l.forEach(f),this.h()},h(){u(s,"type","button"),u(s,"class",i=_(["size--"+a[0],a[0]==="small"?"h6":"p","type--"+a[1]].join(" "))+" svelte-b9w8gh")},m(t,l){y(t,s,l),e&&e.m(s,null),n=!0},p(t,[l]){e&&e.p&&(!n||l&4)&&z(e,o,t,t[2],n?v(o,t[2],l,null):g(t[2]),null),(!n||l&3&&i!==(i=_(["size--"+t[0],t[0]==="small"?"h6":"p","type--"+t[1]].join(" "))+" svelte-b9w8gh"))&&u(s,"class",i)},i(t){n||(B(e,t),n=!0)},o(t){j(e,t),n=!1},d(t){t&&f(s),e&&e.d(t)}}}function S(a,s,i){let{$$slots:n={},$$scope:o}=s,{size:e="default"}=s,{type:t="default"}=s;return a.$$set=l=>{"size"in l&&i(0,e=l.size),"type"in l&&i(1,t=l.type),"$$scope"in l&&i(2,o=l.$$scope)},[e,t,o,n]}class C extends r{constructor(s){super(),m(this,s,S,w,h,{size:0,type:1})}}export{C as B};