<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- PURPOSE: Test two xsl:when's without xsl:otherwise, one succeeding. -->
  <xsl:template match="doc">
    <out>
      <xsl:choose>
        <xsl:when test="blah">2</xsl:when>
        <xsl:when test="foo">1</xsl:when>
      </xsl:choose>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
