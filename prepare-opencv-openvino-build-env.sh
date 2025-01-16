# TODO: make the paths relative
export CGO_CXXFLAGS="--std=c++11"
export CGO_CPPFLAGS="-I/usr/include/ -I/usr/include/ -I/mnt/c/Users/gera/dev/install/include/"
export CGO_LDFLAGS="-I/usr/lib -I/usr/lib64 -I/mnt/c/Users/gera/dev/install/lib/ -lpthread -ldl -lopenvino -lopencv_core -lopencv_videoio -lopencv_imgproc -lopencv_highgui -lopencv_imgcodecs -lopencv_objdetect -lopencv_features2d -lopencv_video -lopencv_dnn -lopencv_calib3d -lopencv_photo"
export PKG_CONFIG_PATH=/usr/lib64/pkgconfig
