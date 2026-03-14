<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test union operator using overlapping node-sets. Results should
       always be output in doc order regardless of order of select attribute. -->
  
<xsl:output indent="no"/>

<xsl:template match="directions">
  <out><xsl:text>
    </xsl:text>
    <xsl:copy-of select="north/* | north/dup1 | north/dup2"/>,
    <xsl:copy-of select="north/dup2 | north/dup1 | north/*"/>,
    <xsl:copy-of select="//north/dup2 | south/preceding-sibling::*[4]/* | north/dup1 | north/*"/>,
    <xsl:copy-of select="north/dup2 | document('select-59.xml')/south/preceding-sibling::*[4]/* | north/*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
