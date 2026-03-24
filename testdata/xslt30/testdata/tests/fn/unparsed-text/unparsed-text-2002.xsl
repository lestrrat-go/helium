<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:transform version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
 
    <!-- test unparsed-text() with one argument 
         using HTTP headers to discover encoding -->

<xsl:output name="test" encoding="iso-8859-1" indent="no"/>
<xsl:template match="/">
  <xsl:result-document format="test">
  <out>
    <utf-8><xsl:value-of select="contains(unparsed-text('https://www.w3.org/Consortium/mission.html'),
                                 'long-term growth')"/></utf-8>
    <iso-8859-1><xsl:value-of select="contains(unparsed-text('https://www.w3.org/TR/1999/WD-font-19990902'),
                                 'Håkon Lie, W3C')"/></iso-8859-1>                                 
  </out> 
  </xsl:result-document> 
</xsl:template>


</xsl:transform>