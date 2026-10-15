function add(a,b){return a+b}
class A { #x=1; get x(){return this.#x} static make(){return new A()} }
async function f(){ try { await 1; throw new Error("e") } catch(e){ return e.message } }
function sw(s){ switch(s){case "a":return 1;case "b":return 2;case "c":return 3;case "d":return 4;default:return 0} }
const o={a:1,b:[1,2,3],c:{d:"x"}};
function* g(){ yield 1; yield 2 }
print(add(1,2), A.make().x, sw("c"), JSON.stringify(o), [...g()].length);
f().then(print);
