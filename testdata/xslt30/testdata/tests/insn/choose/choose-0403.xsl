<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- PURPOSE: Test when by with xsl:otherwise, fail on when. -->
  <xsl:template match="doc">
    <out>
      <xsl:choose>
        <xsl:when test="blah">1</xsl:when>
        <xsl:otherwise>2</xsl:otherwise>
      </xsl:choose>
    </out>
  </xsl:template>
 
</xsl:stylesheet>