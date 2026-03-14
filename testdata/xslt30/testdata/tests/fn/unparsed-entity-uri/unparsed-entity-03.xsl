<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="http://example.com/expression-1903" exclude-result-prefixes="f">

  <!-- PURPOSE: test unparsed-entity-public-id#2  -->
  <?spec xslt#unparsed-entity-public-id?>
  <xsl:template match="doc">
    <out>
      <xsl:sequence select="f:f(.)"/>
    </out>
  </xsl:template>
  
  <xsl:function name="f:f">
    <xsl:param name="doc"/>
      <a><xsl:value-of select="ends-with(unparsed-entity-uri('hatch-pic', $doc), '/fn/grafix/OpenHatch.gif')"/></a>
      <b><xsl:value-of select="unparsed-entity-public-id('hatch-pic', $doc)"/></b>
      <c><xsl:value-of select="ends-with(unparsed-entity-uri('watch-pic', $doc), '/fn/grafix/OpenHatch.gif')"/></c>
      <d><xsl:value-of select="unparsed-entity-public-id('watch-pic', $doc)"/></d>
   </xsl:function>         
 
</xsl:stylesheet>
