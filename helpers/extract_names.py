import onnx,ast
m = onnx.load('/mnt/c/Users/gera/Downloads/yolov7/yolov7-tiny_736x1280.onnx')
props = { p.key : p.value for p in m.metadata_props }
print(m.metadata_props)
if 'names' in props:
    names = ast.literal_eval(props['names'])
    print(names)
