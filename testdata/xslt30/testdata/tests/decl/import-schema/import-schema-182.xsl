<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:ipo="http://www.example.com/PO1"
                exclude-result-prefixes="ipo xs"
                version="2.0">


  <!-- Purpose: Test with typical purchase order schema with simple and complex types.-->
 
   <xsl:import-schema namespace="http://www.example.com/PO1">
      <xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                  xmlns:po="http://www.example.com/PO1"
                  targetNamespace="http://www.example.com/PO1">
         <xsd:element name="purchaseOrder" type="po:PurchaseOrderType"/>
         <xsd:element name="comment" type="xsd:string"/>
         <xsd:complexType name="PurchaseOrderType">
            <xsd:sequence>
               <xsd:element name="shipTo" type="po:USAddress"/>
               <xsd:element name="billTo" type="po:USAddress"/>
               <xsd:element ref="po:comment" minOccurs="0"/>
               <xsd:element name="items" type="po:Items"/>
            </xsd:sequence>
            <xsd:attribute name="orderDate" type="xsd:date"/>
         </xsd:complexType>
         <xsd:complexType name="USAddress">
            <xsd:sequence>
               <xsd:element name="name" type="xsd:string"/>
               <xsd:element name="street" type="xsd:string"/>
               <xsd:element name="city" type="xsd:string"/>
               <xsd:element name="state" type="xsd:string"/>
               <xsd:element name="zip" type="xsd:decimal"/>
            </xsd:sequence>
            <xsd:attribute name="country" type="xsd:NMTOKEN" fixed="US"/>
         </xsd:complexType>
         <xsd:complexType name="Items">
            <xsd:sequence>
               <xsd:element name="item" minOccurs="0" maxOccurs="unbounded">
                  <xsd:complexType>
                     <xsd:sequence>
                        <xsd:element name="productName" type="xsd:string"/>
                        <xsd:element name="quantity">
                           <xsd:simpleType>
                              <xsd:restriction base="xsd:positiveInteger">
                                 <xsd:maxExclusive value="100"/>
                              </xsd:restriction>
                           </xsd:simpleType>
                        </xsd:element>
                        <xsd:element name="USPrice" type="xsd:decimal"/>
                        <xsd:element ref="po:comment" minOccurs="0"/>
                        <xsd:element name="shipDate" type="xsd:date" minOccurs="0"/>
                     </xsd:sequence>
                     <xsd:attribute name="partNum" type="po:SKU" use="required"/>
                  </xsd:complexType>
               </xsd:element>
            </xsd:sequence>
         </xsd:complexType>
         <!-- Stock Keeping Unit, acode for identifying products -->
         <xsd:simpleType name="SKU">
            <xsd:restriction base="xsd:string">
               <xsd:pattern value="\d{3}-[A-Z]{2}"/>
            </xsd:restriction>
         </xsd:simpleType>
      </xsd:schema>
   </xsl:import-schema>

   <xsl:variable name="var" as="element()">
      <ipo:purchaseOrder orderDate="1999-10-20" xsl:type="ipo:PurchaseOrderType">
         <shipTo country="US">
            <name>Alice Smith</name>
            <street>123 Maple Street</street>
            <city>Mill Valley</city>
            <state>CA</state>
            <zip>90952</zip>
         </shipTo>
         <billTo country="US">
            <name>Robert Smith</name>
            <street>8 Oak Avenue</street>
            <city>Old Town</city>
            <state>PA</state>
            <zip>95819</zip>
         </billTo>
         <ipo:comment>Hurry, my lawn is going wild!</ipo:comment>
         <items>
            <item partNum="872-AA">
               <productName>Lawnmower</productName>
               <quantity>1</quantity>
               <USPrice>148.95</USPrice>
               <ipo:comment>Confirm this is electric</ipo:comment>
            </item>
            <item partNum="926-AA">
               <productName>Baby Monitor</productName>
               <quantity>1</quantity>
               <USPrice>39.98</USPrice>
               <shipDate>1999-05-21</shipDate>
            </item>
         </items>
      </ipo:purchaseOrder>
   </xsl:variable>

   <xsl:template match="/doc">
      <out>
         <xsl:value-of select="$var instance of element(*,ipo:PurchaseOrderType) "/>
         <xsl:value-of select="$var instance of schema-element(ipo:purchaseOrder) "/>
      </out>
   </xsl:template>



</xsl:stylesheet>
