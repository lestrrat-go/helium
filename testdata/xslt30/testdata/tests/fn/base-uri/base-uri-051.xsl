<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
             version="3.0" expand-text="yes">
<!-- Purpose: Test base-uri applied to input nodes created using external entities.-->

   <xsl:template match="/">
      <out>
         <one>{base-uri(//para[.='one'])}</one>
         <two>{base-uri(//item[.='two'])}</two>
         <three>{base-uri(//item[.='three'])}</three>
         <four>{base-uri(//item[.='three']/ancestor::para)}</four>
         <five>{base-uri(//para[.='five'])}</five>
      </out>
   </xsl:template>
   
   <xsl:template name="base-uri-051a">
      <xsl:apply-templates select="doc('baseuri051.xml')"/>
   </xsl:template>
   
   <xsl:template name="base-uri-051b">
      <xsl:apply-templates select="document('baseuri051.xml')"/>
   </xsl:template>
 
</xsl:transform>
