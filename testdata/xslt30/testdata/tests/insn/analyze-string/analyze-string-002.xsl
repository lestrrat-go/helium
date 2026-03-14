<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#analyze-string?>
  <!-- PURPOSE: Type-checking of select attribute of xsl:analyze-string. -->
  
  <xsl:param name="re" select="'e'"/>
  <xsl:param name="node" as="element()"><x:out xmlns:x="http://energex.com/"/></xsl:param>
  <xsl:param name="in" select="namespace-uri-from-QName(node-name($node))"/>
  
  <xsl:template match="/">
    <out>
      <xsl:variable name="v">
        <xsl:analyze-string select="$in" regex="{$re}">
         <xsl:matching-substring>[<xsl:value-of select="."/>]</xsl:matching-substring>
         <xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring>
        </xsl:analyze-string>
      </xsl:variable>
      <xsl:value-of select="translate($v, '&#xa;', '|')"/>
    </out>
  </xsl:template>

</xsl:stylesheet>