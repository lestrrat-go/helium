<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- DataManipulation001 in NIST suite -->
  <!-- Purpose: Test xsl:if inside xsl:when. -->
  <!-- Also (MHK) tests xml:space on xsl:choose -->

<xsl:template match="doc">
  <out>
    <xsl:choose xml:space="preserve">
      <xsl:when test = "2 &gt; 1">
        <xsl:if test = "9 mod 3 = 0">
          <xsl:text>Test executed successfully.</xsl:text>
        </xsl:if>
      </xsl:when>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
