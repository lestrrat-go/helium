<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
<!-- test a composite attribute set that references the static context -->

   <xsl:attribute-set name="b" xml:base="http://www.example.com/one">
      <xsl:attribute name="base-one" select="static-base-uri()"/>
   </xsl:attribute-set> 

   <xsl:attribute-set name="b" xml:base="http://www.example.com/two">
     <xsl:attribute name="base-two" select="static-base-uri()"/>
   </xsl:attribute-set> 
   
   <xsl:template match="/">
     <a xsl:use-attribute-sets="b"/>
   </xsl:template>

</xsl:transform>

