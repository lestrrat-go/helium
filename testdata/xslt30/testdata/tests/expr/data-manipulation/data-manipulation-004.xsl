<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
  <!-- Purpose: Test an "xsl:if" with a complex false 
       expression. -->
    
<xsl:template match="doc">
  <out>
    <xsl:if test = "(((((2*10)-4)+9) div 5) div 2) > 4">
        Test do not excuted successfully!!
    </xsl:if>
    This text (and the "out" text) should be the only text 
    displayed by this test.
  </out>
</xsl:template>

</xsl:stylesheet>
