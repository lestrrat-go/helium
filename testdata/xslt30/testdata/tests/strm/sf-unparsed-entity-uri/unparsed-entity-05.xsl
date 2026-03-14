<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: unparsed entities available during document node processing  -->

  <xsl:mode streamable="yes"/>
  <xsl:template match="/">
    <out>
        <a><xsl:value-of select="ends-with(unparsed-entity-uri('hatch-pic'), '/strm/grafix/OpenHatch.gif')"/></a>
        <b><xsl:value-of select="unparsed-entity-public-id('hatch-pic')"/></b>
        <c><xsl:value-of select="ends-with(unparsed-entity-uri('watch-pic'), '/strm/grafix/OpenHatch.gif')"/></c>
        <d><xsl:value-of select="unparsed-entity-public-id('watch-pic')"/></d>
        <xsl:copy-of select="*"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
