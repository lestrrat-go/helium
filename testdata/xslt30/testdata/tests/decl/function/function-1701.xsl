<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

 <xsl:template match="/">
  <out>
   <xsl:apply-templates/>
  </out>
 </xsl:template>

 <xsl:template match="doc">
  <xsl:for-each select="a">
   <a>
    <xsl:value-of select="format-integer(. ,'1','de' cast as xs:language)"/>
   </a>
  </xsl:for-each>
  <xsl:for-each select="a">
   <a>
    <xsl:value-of select="format-integer(. ,'Ww')"/>
   </a>
  </xsl:for-each>
  <xsl:for-each select="a">
   <a>
    <xsl:value-of select="format-integer(. ,'A')"/>
   </a>
  </xsl:for-each>
 </xsl:template>
 
 <xsl:strip-space elements="*"/>
</xsl:stylesheet>
