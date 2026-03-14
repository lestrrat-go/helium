<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test numbering of fractional value argument  -->
  <xsl:template match="doc">
    <out>
        10 = <xsl:number value="10" format="01" />;
        3.6 = <xsl:number value="3.6" format="01" />;
        0 = <xsl:number value="0" format="01" />;
        0.3 = <xsl:number value="0.3" format="01"  />;
        0.7 = <xsl:number value="0.7" format="01" />;
        200 div 3 = <xsl:number value="200 div 3" format="01" />;
    </out>
  </xsl:template>
 
</xsl:stylesheet>
