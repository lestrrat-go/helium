<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xsd="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xsd" version="2.0">


   <!-- Purpose: Test with xsl:import-schema without @namespace which contains an xs:schema 
				element without @targetNamespace. Verify that elements in no namespace from 
				the imported schema are available for use in the stylesheet. -->

   <xsl:import-schema>
      <xsd:schema>
         <xsd:element name="doc-inline" type="docType-inline"/>
         <xsd:element name="de1-decimal-enumeration-inline" type="de1-decimal-enumeration-Inline"/>
         <xsd:element name="de2-decimal-maxExclusive-inline" type="de2-decimal-maxExclusive-Inline"/>
         <xsd:element name="de5-decimal-fractionDigits-totalDigits-inline"
            type="de5-decimal-fractionDigits-totalDigits-Inline"/>
         <xsd:attribute name="attr1" type="de1-decimal-enumeration-Inline"/>
         <xsd:attribute name="attr2" type="de2-decimal-maxExclusive-Inline"/>
         <xsd:complexType name="docType-inline">
            <xsd:sequence>
               <xsd:element name="elem-attr-inline" minOccurs="0" maxOccurs="1">
                  <xsd:complexType>
                     <xsd:attribute ref="attr1" use="optional"/>
                     <xsd:attribute ref="attr2" use="optional"/>
                  </xsd:complexType>
               </xsd:element>
               <xsd:element ref="de1-decimal-enumeration-inline" minOccurs="0" maxOccurs="1"/>
               <xsd:element ref="de2-decimal-maxExclusive-inline" minOccurs="0" maxOccurs="1"/>
               <xsd:element ref="de5-decimal-fractionDigits-totalDigits-inline" minOccurs="0"
                  maxOccurs="1"/>
            </xsd:sequence>
         </xsd:complexType>
         <!-- Following are types derived from decimal -->
         <xsd:simpleType name="de1-decimal-enumeration-Inline">
            <xsd:restriction base="xsd:decimal">
               <xsd:enumeration value="1"/>
               <xsd:enumeration value="-1"/>
               <xsd:enumeration value="0"/>
               <xsd:enumeration value="1.1"/>
               <xsd:enumeration value="-1.1"/>
               <xsd:enumeration value="1.01"/>
               <xsd:enumeration value="-1.01"/>
               <xsd:enumeration value="1.001"/>
               <xsd:enumeration value="-1.001"/>
               <xsd:enumeration value="1.0001"/>
               <xsd:enumeration value="-1.0001"/>
               <xsd:enumeration value="1.0101"/>
               <xsd:enumeration value="-1.0101"/>
               <xsd:enumeration value="11111"/>
               <xsd:enumeration value="-11111"/>
            </xsd:restriction>
         </xsd:simpleType>
         <xsd:simpleType name="de2-decimal-maxExclusive-Inline">
            <xsd:restriction base="xsd:decimal">
               <xsd:maxExclusive value="123.456789"/>
            </xsd:restriction>
         </xsd:simpleType>
         <xsd:simpleType name="de5-decimal-fractionDigits-totalDigits-Inline">
            <xsd:restriction base="xsd:decimal">
               <xsd:totalDigits value="11"/>
               <xsd:fractionDigits value="4"/>
            </xsd:restriction>
         </xsd:simpleType>
      </xsd:schema>
   </xsl:import-schema>

   <xsl:variable name="var1" as="de1-decimal-enumeration-Inline*">
      <xsl:element name="de1-decimal-enumeration-inline">1.1</xsl:element>
      <xsl:element name="de1-decimal-enumeration-inline">1.0001</xsl:element>
   </xsl:variable>

   <xsl:variable name="var2" as="element()">
      <doc-inline xsl:type="docType-inline">
         <de5-decimal-fractionDigits-totalDigits-inline>1122334.4556</de5-decimal-fractionDigits-totalDigits-inline>
      </doc-inline>
   </xsl:variable>

   <xsl:variable name="var3" as="attribute()">
      <xsl:attribute name="attr2" type="de2-decimal-maxExclusive-Inline">123.456</xsl:attribute>
   </xsl:variable>

   <xsl:template match="/doc">
      <out>
         <type>
            <xsl:value-of select="sum($var1)"/>
         </type>
         <elem>
            <xsl:value-of select="$var2 instance of element(*, docType-inline)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var2 instance of schema-element(doc-inline)"/>
            <xsl:copy-of select="$var2"/>
         </elem>
         <attr>
            <xsl:value-of
               select="$var3 instance of attribute(attr2, de2-decimal-maxExclusive-Inline)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var3 instance of schema-attribute(attr2)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var3"/>
         </attr>
      </out>
   </xsl:template>



</xsl:stylesheet>
