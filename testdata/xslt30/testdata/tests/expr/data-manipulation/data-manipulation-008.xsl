<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- Purpose: Test an "xsl:choose" with a function as part of the
       test (use "round" function). -->
    
<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test = "(round(3.7) > 3)">
        Test executed successfully!!
      </xsl:when>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
