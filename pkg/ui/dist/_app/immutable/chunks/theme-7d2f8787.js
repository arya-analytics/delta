const o={Key:"arya-base",Colors:{White:"#FFFFFF",Black:"#2E2E2E",Primary:{M1:"#3363BE",Z:"#3774D0",P1:"#3C86E3"},Gray:{M2:"#66676C",M1:"#798187",Z:"#ACB5BD",P1:"#DDE2E5",P2:"#BDBDBD"},Error:{M1:"#CF1322",Z:"#F5222D",P1:"#FF4547"},Text:"#2E2E2E",Background:"#FFFFFF"},Dimensions:{Grid:6,BorderRadius:2,BorderWidth:1}},e={...o,Key:"arya-light"};o.Colors.White,o.Colors.Black,o.Colors.Primary,o.Colors.Gray.P2,o.Colors.Gray.P1,o.Colors.Gray.Z,o.Colors.Gray.M1,o.Colors.Gray.M2,o.Colors.White,o.Colors.Error,o.Colors.Black,o.Dimensions;const y=r=>r+"px",l=(r,s)=>{s.style.setProperty("--white",r.Colors.White),s.style.setProperty("--black",r.Colors.Black),s.style.setProperty("--primary-m1",r.Colors.Primary.M1),s.style.setProperty("--primary-z",r.Colors.Primary.Z),s.style.setProperty("--primary-p1",r.Colors.Primary.P1),s.style.setProperty("--gray-m2",r.Colors.Gray.M2),s.style.setProperty("--gray-m1",r.Colors.Gray.M1),s.style.setProperty("--gray-z",r.Colors.Gray.Z),s.style.setProperty("--gray-p1",r.Colors.Gray.P1),s.style.setProperty("--gray-p2",r.Colors.Gray.P2),s.style.setProperty("--error-m1",r.Colors.Error.M1),s.style.setProperty("--error-z",r.Colors.Error.Z),s.style.setProperty("--error-p1",r.Colors.Error.P1),s.style.setProperty("--background",r.Colors.Background),s.style.setProperty("--text-color",r.Colors.Text),s.style.setProperty("--grid",y(r.Dimensions.Grid)),s.style.setProperty("--border-radius",y(r.Dimensions.BorderRadius)),s.style.setProperty("--border-width",y(r.Dimensions.BorderWidth))},a=[e,e];export{l as a,a as b};