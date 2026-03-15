<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform 
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
   xml:base="http://www.example.org/"
   version="2.0">
   
   <!-- Purpose: Test with base-uri($arg) where $arg is a copied single element node (no parent).-->

   <xsl:template match="/doc">
      <out>
         <xsl:apply-templates select="str1/*"/>
      </out>
   </xsl:template>

   <xsl:template match="substring1">
      <xsl:variable name="elemcopy" as="element()" xml:base="/main/">
         <xsl:copy>new content</xsl:copy>
      </xsl:variable>

      <xsl:value-of select="base-uri($elemcopy)"/>

   </xsl:template>
</xsl:transform>
