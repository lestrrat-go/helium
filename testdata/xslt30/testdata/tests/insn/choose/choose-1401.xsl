<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- DataManipulation002 in NIST suite -->
  <!-- Purpose: Test xsl:if inside xsl:otherwise. -->

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test = "2 &gt; 3">
        <xsl:text>Test failed!!</xsl:text>
      </xsl:when>
      <xsl:otherwise>
        <xsl:if test = "9 mod 3 = 0">
          <xsl:text>Test executed successfully.</xsl:text>
        </xsl:if>
      </xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
