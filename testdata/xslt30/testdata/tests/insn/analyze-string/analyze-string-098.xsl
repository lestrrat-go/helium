<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:e="http://example.org">
 <xsl:param name="delimiter" select="'\|'"/>

 <xsl:variable name="data">
  <row>a|b|de|fg|hijkl|vw</row>
  <row>a|b|ss|fg|hijkl|vw</row>
  <row>a|b|ff|fg|hijkl|vw</row>
  <row>a|b|hl|fg|hijkl|vw</row>
  <row>a|b|mm|fg|hijkl|vw</row>
  <row>a|b|oo|fg|hijkl|vw</row>
 </xsl:variable>

 <xsl:template match="/" name="main">
  <out>
   <xsl:for-each select="$data/row">
    <row>
    <xsl:analyze-string select="." regex="{$delimiter}">
     <xsl:non-matching-substring>
      <cell><xsl:value-of select="."/></cell>
     </xsl:non-matching-substring>
    </xsl:analyze-string>
    </row>
   </xsl:for-each>
  </out>
 </xsl:template>
 
</xsl:stylesheet>
