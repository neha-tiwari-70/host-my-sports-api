-- name: create-states-table
CREATE TABLE IF NOT EXISTS states (
    id INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    country_id INT NOT NULL 
);

-- name: insert-states
INSERT INTO states (id , name, country_id) VALUES
(1, 'Andhra Pradesh', 105),
(2, 'Assam', 105),
(3, 'Arunachal Pradesh', 105),
(4, 'Bihar', 105),
(5, 'Gujarat', 105),
(6, 'Haryana', 105),
(7, 'Himachal Pradesh', 105),
(8, 'Jammu & Kashmir', 105),
(9, 'Karnataka', 105),
(10, 'Kerala', 105),
(11, 'Madhya Pradesh', 105),
(12, 'Maharashtra', 105),
(13, 'Manipur', 105),
(14, 'Meghalaya', 105),
(15, 'Mizoram', 105),
(16, 'Nagaland', 105),
(17, 'Orissa', 105),
(18, 'Punjab', 105),
(19, 'Rajasthan', 105),
(20, 'Sikkim', 105),
(21, 'Tamil Nadu', 105),
(22, 'Tripura', 105),
(23, 'Uttar Pradesh', 105),
(24, 'West Bengal', 105),
(25, 'Delhi', 105),
(26, 'Goa', 105),
(27, 'Pondicherry', 105),
(28, 'Lakshwadeep', 105),
(29, 'Daman & Diu', 105),
(30, 'Dadra & Nagar', 105),
(31, 'Chandigrah', 105),
(32, 'Andaman & Nicobar', 105),
(33, 'Uttranchal', 105),
(34, 'Jharkhand', 105),
(35, 'Chattisgarh', 105);