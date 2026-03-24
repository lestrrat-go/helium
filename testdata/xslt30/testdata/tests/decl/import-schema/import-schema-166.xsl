<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:foo="http://ns.example.com/sch002"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="foo xs"
                version="2.0">

  <!-- Purpose: Simple test of xsl:import-schema with @namespace and @schema-location.Verify that top-level 
  				element and attribute declarations and defined types that are in  @namespace are available 
  				for use in the stylesheet.-->
 
   <xsl:import-schema namespace="http://ns.example.com/sch002"
                      schema-location="schema002.xsd"/>

   <xsl:variable name="var1" as="foo:testType*">
      <xsl:sequence select="(foo:testType(2000),foo:testType(2001),foo:testType(2002),foo:testType(2003))"/>
   </xsl:variable>

   <xsl:variable name="var2" as="element()">
      <foo:complexTest xsl:type="foo:complexTestType" foo:type="333">
         <subject>math</subject>
         <size>50</size>
      </foo:complexTest>
   </xsl:variable>

   <xsl:variable name="var3" as="attribute()">
	     <xsl:attribute name="foo:type" type="foo:testType">444</xsl:attribute>
   </xsl:variable>

   <xsl:template match="/doc">
      <xsl:text>
</xsl:text>
      <out>
         <xsl:text>
</xsl:text>
         <type>
            <xsl:value-of select="sum($var1)"/>
         </type>
         <xsl:text>
</xsl:text>
         <elem>
            <xsl:value-of select="$var2 instance of schema-element(foo:complexTest)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var2/@foo:type instance of schema-attribute(foo:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$var2"/>
         </elem>
         <xsl:text>
</xsl:text>
         <attrib>
            <xsl:value-of select="$var3 instance of schema-attribute(foo:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var3"/>
         </attrib>
         <xsl:text>
</xsl:text>
      </out>
   </xsl:template>


</xsl:stylesheet>
