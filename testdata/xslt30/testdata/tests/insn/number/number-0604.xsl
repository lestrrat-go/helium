<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test negative number formatting -->
  <xsl:template match="doc">
    <out>
           <xsl:number value="-99.83" format="[0001]"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
