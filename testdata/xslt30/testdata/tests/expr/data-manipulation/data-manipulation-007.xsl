<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
  <!-- Purpose: Test an "xsl:if" with a function as part of the
       test (use "round" function). -->
    
<xsl:template match="doc">
  <out>
     <xsl:if test = "(round(3.7) > 3)">
       Test executed successfully!!
     </xsl:if>
  </out>
</xsl:template>

</xsl:stylesheet>
