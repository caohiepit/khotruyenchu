# Client UI - Tiên Kiếm Tông (Comics & Novel Reader)

Giao diện người dùng đọc truyện chữ phong cách **Tu Tiên / Huyền Huyễn** kết hợp hệ thống **Shopee Affiliate** chuyển đổi cao.

## 🌐 Đường Dẫn Truy Cập
- **Site Client (Trang Đọc Truyện)**: `http://localhost:8000/` (hoặc `http://localhost:8000/client/`)
- **Site Admin (Cổng Quản Trị)**: `http://localhost:8000/admin/`

## 🎨 Kiến Trúc Giao Diện (Theo Bộ Thiết Kế Chuẩn)
1. **Trang Chủ (`#/` - [01_trang_chu_homepage.jpg](../../designs/01_trang_chu_homepage.jpg))**:
   - **Hero Banner Đỉnh Phong**: Slider tự động chuyển cảnh các bộ tiểu thuyết nổi bật kèm hiệu ứng hào quang kiếm khí và nút **ĐỌC NGAY**.
   - **Bảng Xếp Hạng Tu Tiên & Thống Kê Tu Luyện**: Xếp hạng Top 5 truyện nhiều chương/lượt xem nhất và tự động tính toán cảnh giới của người đọc dựa trên số chương đã đọc.
   - **Tiểu Thuyết Nổi Bật & Huy Hiệu Cảnh Giới**: Gắn huy hiệu cảnh giới phát sáng (`LUYỆN KHÍ`, `TRÚC CƠ`, `KIM ĐAN`, `NGUYÊN ANH`, `ĐẠI THỪA KỲ`) cùng bộ lọc nhanh theo thể loại.
   - **Banner Tàng Bảo Các**: Khu vực mở kho pháp bảo Shopee Affiliate cuối trang và modal săn ưu đãi.

2. **Trang Chi Tiết Truyện (`#/novel/:slug` - [02_chi_tiet_truyen_detail.jpg](../../designs/02_chi_tiet_truyen_detail.jpg))**:
   - **Khung Bìa Phát Hào Quang**: Nút **ĐỌC TỪ ĐẦU** mạ vàng và nút **ĐỌC TIẾP** tự động nhớ chương đang đọc dở, kèm thanh chỉ số 5 cột (`Rating`, `Views`, `Reviews`, `Chapters`, `Trạng Thái`).
   - **Mục Lục Chương Gắn Icon Cảnh Giới**: Tìm kiếm nhanh theo số chương/tiêu đề, đảo chiều sắp xếp (`Cũ → Mới` / `Mới → Cũ`), phân trang mượt mà.
   - **Sidebar Linh Bảo Quán**: Cố định bên phải hiển thị danh sách vật phẩm phong thủy, trà đạo, bàn phím cơ... điều hướng trực tiếp sang Shopee.

3. **Trình Đọc Chương (`#/read/:slug/:chapterId` - [03_trinh_doc_truyen_reader.jpg](../../designs/03_trinh_doc_truyen_reader.jpg))**:
   - **Trải Nghiệm Đọc Đa Nền Tảng**: Hỗ trợ đổi cỡ chữ (`A-` / `A+`), chuyển đổi 3 chủ đề nền (`Huyền Mặc` tối, `Cổ Thư` vàng giấy, `Tinh Khiết` sáng), hỗ trợ phím mũi tên `←` / `→` để chuyển chương.
   - **Box Phúc Địa Tu Tiên - Bảo Vật Khuyên Dùng**: Hộp gợi ý vật phẩm Shopee thiết kế tự nhiên ngay cạnh/cuối nội dung chương.
   - **Tự Động Lưu Tiến Độ (`Tủ Truyện Tu Luyện`)**: Lưu lịch sử chương đã đọc vào `localStorage`, đồng bộ với mục **TU LUYỆN** trên thanh điều hướng.
