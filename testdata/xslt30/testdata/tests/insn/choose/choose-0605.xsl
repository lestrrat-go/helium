<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- DataManipulation008 in NIST suite -->
  <!-- Purpose: Test a function (round) in the test attribute. -->

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test = "(round(3.7) &gt; 3)">
        <xsl:text>Test executed successfully</xsl:text>
      </xsl:when>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
