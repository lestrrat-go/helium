<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


   <!-- Purpose: Test with xsl:import-schema without @namespace to show that top-level 
  				element and attribute declarations and defined types that are in no 
  				namespace are available for use in the stylesheet.@schema-location 
  				is present. -->

   <xsl:import-schema schema-location="schema176a.xsd"/>

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
