package graphics

const VertexShaderSource = `
    #version 410
    layout(location = 0) in vec3 vp;
    uniform mat4 projection;
    uniform mat4 view;
    uniform mat4 model;
    void main() {
        gl_Position = projection * view * model * vec4(vp, 1.0);
    }
` + "\x00"

const FragmentShaderSource = `
    #version 410
    out vec4 frag_colour;
    uniform vec4 color;
    void main() {
        frag_colour = color;
    }
` + "\x00"
