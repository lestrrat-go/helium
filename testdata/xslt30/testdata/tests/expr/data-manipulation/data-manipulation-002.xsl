<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- Purpose: Test an "xsl:choose" instruction with and "xsl:if"
       instruction as part of its content template (if inside
       "otherwise". -->

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test = "2 > 3">
         Test not executed successfully!!
      </xsl:when>
      <xsl:otherwise>
       <xsl:if test = "9 mod 3 = 0">
         Test excuted successfully!!
       </xsl:if>
      </xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
