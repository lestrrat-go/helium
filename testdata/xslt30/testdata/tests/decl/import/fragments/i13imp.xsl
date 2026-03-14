<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:template match="author">
  IMPORT author: <xsl:value-of select="."/><xsl:text>&#10;</xsl:text>
</xsl:template>

<xsl:template match="overview">
  <xsl:text>IMPORT overview: </xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
