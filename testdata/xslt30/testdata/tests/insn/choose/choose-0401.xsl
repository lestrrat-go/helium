<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- PURPOSE: Test xsl:when by itself, success. -->
  <xsl:template match="doc">
    <out>
      <xsl:choose>
        <xsl:when test="foo">1</xsl:when>
      </xsl:choose>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
